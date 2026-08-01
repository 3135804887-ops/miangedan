// Package ingestion implements region-scoped resume quarantine uploads and file-safety scanning.
//
// Tracking: TASK-012; PRD-001 FR-001, FR-006; TM-02; SEC-020; NFR-006.
package ingestion

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"miangedan/services/region"
)

// MaxResumeBytes is the hard upper bound for a resume upload.
const MaxResumeBytes int64 = 10 * 1024 * 1024

// UploadStatus is the durable lifecycle state of an uploaded resume.
type UploadStatus string

const (
	// StatusQuarantined means the object is isolated and awaiting a scan result.
	StatusQuarantined UploadStatus = "QUARANTINED"
	// StatusScanning means an attested sandbox scan is running.
	StatusScanning UploadStatus = "SCANNING"
	// StatusAccepted means the object passed all security checks.
	StatusAccepted UploadStatus = "ACCEPTED"
	// StatusRejected means the object failed security checks and was deleted.
	StatusRejected UploadStatus = "REJECTED"
	// StatusRetryableFailed means the object is retained after a transient scan failure.
	StatusRetryableFailed UploadStatus = "RETRYABLE_FAILURE"
)

// RejectionReason is a stable, user-understandable security rejection code.
type RejectionReason string

const (
	// ReasonUnsupportedType rejects file extensions outside PDF, DOC, and DOCX.
	ReasonUnsupportedType RejectionReason = "unsupported_type"
	// ReasonOversized rejects content exceeding the 10 MiB limit.
	ReasonOversized RejectionReason = "oversized"
	// ReasonTypeSpoofed rejects files whose signature contradicts the extension.
	ReasonTypeSpoofed RejectionReason = "type_spoofed"
	// ReasonCorrupted rejects malformed or structurally incomplete documents.
	ReasonCorrupted RejectionReason = "corrupted"
	// ReasonEncrypted rejects encrypted documents that cannot be inspected safely.
	ReasonEncrypted RejectionReason = "encrypted"
	// ReasonVirusDetected rejects content identified by the malware detector.
	ReasonVirusDetected RejectionReason = "virus_detected"
	// ReasonMacrosDetected rejects active macro or script content.
	ReasonMacrosDetected RejectionReason = "macros_detected"
	// ReasonArchiveBombDetected rejects unsafe archive expansion characteristics.
	ReasonArchiveBombDetected RejectionReason = "archive_bomb_detected"
	// ReasonSandboxPolicyViolated rejects execution without the required isolation.
	ReasonSandboxPolicyViolated RejectionReason = "sandbox_policy_violation"
)

var (
	// ErrIdempotencyConflict indicates key reuse with different immutable input.
	ErrIdempotencyConflict = errors.New("idempotency key reused with different upload content")
	// ErrNotRetryable indicates that the current state cannot be retried.
	ErrNotRetryable = errors.New("upload scan is not retryable")
	// ErrScanTimeout marks a transient sandbox timeout.
	ErrScanTimeout = errors.New("sandbox scan timed out")
	// ErrScannerUnavailable marks a transient scanner outage.
	ErrScannerUnavailable = errors.New("sandbox scanner unavailable")
)

// FileKind is the file type established from content signatures.
type FileKind string

const (
	// FilePDF is a PDF document.
	FilePDF FileKind = "pdf"
	// FileDOC is a legacy OLE Word document.
	FileDOC FileKind = "doc"
	// FileDOCX is an Open Packaging Convention Word document.
	FileDOCX FileKind = "docx"
)

// MediaType returns the canonical media type for the detected kind.
func (k FileKind) MediaType() string {
	switch k {
	case FilePDF:
		return "application/pdf"
	case FileDOC:
		return "application/msword"
	case FileDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return ""
	}
}

// UploadImpact describes retention, retry, billing, and scoring consequences for users.
type UploadImpact struct {
	OriginalInputRetained bool
	Retryable             bool
	Billable              bool
	ScoringAffected       bool
	RetryAction           string
}

// SandboxAttestation records the security properties asserted for one-shot scanning.
type SandboxAttestation struct {
	NetworkDisabled        bool
	Ephemeral              bool
	ReadOnlyRootFilesystem bool
	CredentialsMounted     bool
}

// Validate fails closed unless every required sandbox property is present.
func (a SandboxAttestation) Validate() error {
	if !a.NetworkDisabled || !a.Ephemeral || !a.ReadOnlyRootFilesystem || a.CredentialsMounted {
		return fmt.Errorf("sandbox must be ephemeral with network disabled, read-only root filesystem, and no credential mounts")
	}
	return nil
}

// ObjectRef identifies an internal object in a regional uploads bucket.
type ObjectRef struct {
	Bucket string
	Key    string
}

// UploadObjectStore deliberately exposes uploads-bucket-only operations. There is no method capable
// of writing exports or media, making three-bucket isolation structural rather than conventional.
type UploadObjectStore interface {
	PutQuarantine(ctx context.Context, dataRegion, key string, content []byte) (ObjectRef, error)
	ReadQuarantine(ctx context.Context, ref ObjectRef) ([]byte, error)
	PromoteAccepted(ctx context.Context, ref ObjectRef) (ObjectRef, error)
	Delete(ctx context.Context, ref ObjectRef) error
}

// ScanRequest contains the isolated bytes and sanitized basename to inspect.
type ScanRequest struct {
	Filename string
	Content  []byte
}

// ScanReport contains either a detected file kind or an explicit rejection.
type ScanReport struct {
	Kind            FileKind
	RejectionReason RejectionReason
	Message         string
}

// SandboxRunner executes scanning inside an attested one-shot environment.
type SandboxRunner interface {
	Attestation() SandboxAttestation
	Scan(ctx context.Context, request ScanRequest) (ScanReport, error)
}

// Upload is the durable aggregate for one resume upload and its security result.
type Upload struct {
	UploadID           string
	UserID             string
	DataRegion         string
	IdempotencyKey     string
	ContentFingerprint [32]byte
	Filename           string
	SizeBytes          int64
	DetectedMediaType  string
	Status             UploadStatus
	ObjectRef          *ObjectRef
	RejectionReason    RejectionReason
	Message            string
	Impact             UploadImpact
	SandboxAttestation *SandboxAttestation
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Repository persists uploads with atomic upload and retry idempotency semantics.
type Repository interface {
	CreateOrGet(ctx context.Context, candidate Upload) (upload Upload, created bool, err error)
	Get(ctx context.Context, uploadID string) (Upload, error)
	Update(ctx context.Context, upload Upload) error
	BeginRetry(ctx context.Context, uploadID, idempotencyKey string) (upload Upload, started bool, err error)
}

// UploadRequest is a validated resume-upload command from an authenticated user.
type UploadRequest struct {
	UserID         string
	DataRegion     string
	IdempotencyKey string
	Filename       string
	DeclaredSize   int64
	Content        []byte
}

// IDGenerator returns an opaque upload identifier.
type IDGenerator func() string

// Service coordinates quarantine storage, attested scanning, and lifecycle persistence.
type Service struct {
	repository Repository
	objects    UploadObjectStore
	sandbox    SandboxRunner
	newID      IDGenerator
	now        func() time.Time
}

// NewService constructs a service only when all adapters and sandbox controls are present.
func NewService(repository Repository, objects UploadObjectStore, sandbox SandboxRunner, newID IDGenerator) (*Service, error) {
	if repository == nil || objects == nil || sandbox == nil || newID == nil {
		return nil, errors.New("repository, uploads object store, sandbox, and id generator are required")
	}
	if err := sandbox.Attestation().Validate(); err != nil {
		return nil, fmt.Errorf("sandbox policy rejected (TASK-012/SEC-020): %w", err)
	}
	return &Service{repository: repository, objects: objects, sandbox: sandbox, newID: newID, now: time.Now}, nil
}

// Upload stores a resume in quarantine and returns its idempotent security outcome.
func (s *Service) Upload(ctx context.Context, request UploadRequest) (Upload, error) {
	if err := validateUploadRequest(request); err != nil {
		return Upload{}, err
	}
	now := s.now().UTC()
	candidate := Upload{
		UploadID:           s.newID(),
		UserID:             request.UserID,
		DataRegion:         request.DataRegion,
		IdempotencyKey:     request.IdempotencyKey,
		ContentFingerprint: sha256.Sum256(request.Content),
		Filename:           filepath.Base(request.Filename),
		SizeBytes:          int64(len(request.Content)),
		Status:             StatusQuarantined,
		Impact:             safeImpact(false, false, ""),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	upload, created, err := s.repository.CreateOrGet(ctx, candidate)
	if err != nil || !created {
		return upload, err
	}
	key := fmt.Sprintf("quarantine/%s/%s/%s", request.UserID, upload.UploadID, candidate.Filename)
	ref, err := s.objects.PutQuarantine(ctx, request.DataRegion, key, request.Content)
	if err != nil {
		return Upload{}, fmt.Errorf("store quarantine upload: %w", err)
	}
	upload.ObjectRef = &ref
	if err := s.repository.Update(ctx, upload); err != nil {
		return Upload{}, fmt.Errorf("persist quarantine object reference: %w", err)
	}
	return s.scan(ctx, upload, request.Content)
}

// Get returns the current upload lifecycle state.
func (s *Service) Get(ctx context.Context, uploadID string) (Upload, error) {
	return s.repository.Get(ctx, uploadID)
}

// Retry rereads and scans the retained quarantine object with idempotent side effects.
func (s *Service) Retry(ctx context.Context, uploadID, idempotencyKey string) (Upload, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return Upload{}, errors.New("Idempotency-Key is required")
	}
	upload, started, err := s.repository.BeginRetry(ctx, uploadID, idempotencyKey)
	if err != nil || !started {
		return upload, err
	}
	if upload.ObjectRef == nil || !upload.Impact.OriginalInputRetained {
		return Upload{}, errors.New("retryable upload lost quarantine reference; fail closed")
	}
	content, err := s.objects.ReadQuarantine(ctx, *upload.ObjectRef)
	if err != nil {
		return Upload{}, fmt.Errorf("read retained quarantine object: %w", err)
	}
	if sha256.Sum256(content) != upload.ContentFingerprint {
		return Upload{}, errors.New("retained quarantine object fingerprint mismatch; fail closed")
	}
	return s.scan(ctx, upload, content)
}

func (s *Service) scan(ctx context.Context, upload Upload, content []byte) (Upload, error) {
	upload.Status = StatusScanning
	upload.UpdatedAt = s.now().UTC()
	if err := s.repository.Update(ctx, upload); err != nil {
		return Upload{}, err
	}
	report, err := s.sandbox.Scan(ctx, ScanRequest{Filename: upload.Filename, Content: content})
	attestation := s.sandbox.Attestation()
	upload.SandboxAttestation = &attestation
	upload.UpdatedAt = s.now().UTC()
	if err != nil {
		if errors.Is(err, ErrScanTimeout) || errors.Is(err, ErrScannerUnavailable) {
			upload.Status = StatusRetryableFailed
			upload.Message = "安全扫描暂时未完成；隔离原件已保留，可重试扫描。未计费且不影响评分。"
			upload.Impact = safeImpact(true, true, fmt.Sprintf("POST /v1/uploads/%s:retry", upload.UploadID))
			if updateErr := s.repository.Update(ctx, upload); updateErr != nil {
				return Upload{}, errors.Join(err, updateErr)
			}
			return upload, nil
		}
		return Upload{}, fmt.Errorf("sandbox scan: %w", err)
	}
	upload.DetectedMediaType = report.Kind.MediaType()
	if report.RejectionReason != "" {
		if upload.ObjectRef != nil {
			if deleteErr := s.objects.Delete(ctx, *upload.ObjectRef); deleteErr != nil {
				return Upload{}, fmt.Errorf("delete rejected quarantine object: %w", deleteErr)
			}
		}
		upload.Status = StatusRejected
		upload.ObjectRef = nil
		upload.RejectionReason = report.RejectionReason
		upload.Message = report.Message
		upload.Impact = safeImpact(false, false, "")
		if err := s.repository.Update(ctx, upload); err != nil {
			return Upload{}, err
		}
		return upload, nil
	}
	if upload.ObjectRef == nil {
		return Upload{}, errors.New("accepted upload has no quarantine object; fail closed")
	}
	accepted, err := s.objects.PromoteAccepted(ctx, *upload.ObjectRef)
	if err != nil {
		return Upload{}, fmt.Errorf("promote accepted upload: %w", err)
	}
	upload.Status = StatusAccepted
	upload.ObjectRef = &accepted
	upload.Message = "文件安全检查通过，可进入简历解析。"
	upload.Impact = safeImpact(true, false, "")
	if err := s.repository.Update(ctx, upload); err != nil {
		return Upload{}, err
	}
	return upload, nil
}

func validateUploadRequest(request UploadRequest) error {
	if strings.TrimSpace(request.UserID) == "" {
		return errors.New("authenticated user ID is required")
	}
	if strings.ContainsAny(request.UserID, "/\\") {
		return errors.New("authenticated user ID contains a path separator")
	}
	if err := region.ValidateDataRegion(request.DataRegion); err != nil {
		return err
	}
	if len(request.IdempotencyKey) < 8 || len(request.IdempotencyKey) > 128 {
		return errors.New("Idempotency-Key length must be 8..128")
	}
	if filepath.Base(request.Filename) != request.Filename || strings.TrimSpace(request.Filename) == "" {
		return errors.New("filename must be a basename")
	}
	if request.DeclaredSize < 0 || request.DeclaredSize > MaxResumeBytes || int64(len(request.Content)) > MaxResumeBytes {
		return NewUploadRejectedError(ReasonOversized, "文件超过 10 MiB 上限，未保存；请压缩或导出较小的 PDF、DOC 或 DOCX 后重试。")
	}
	if request.DeclaredSize != int64(len(request.Content)) {
		return errors.New("declared size does not match actual content length")
	}
	switch strings.ToLower(filepath.Ext(request.Filename)) {
	case ".pdf", ".doc", ".docx":
	default:
		return NewUploadRejectedError(ReasonUnsupportedType, "仅支持 PDF、DOC、DOCX 简历；该文件未保存。")
	}
	return nil
}

func safeImpact(retained, retryable bool, action string) UploadImpact {
	return UploadImpact{
		OriginalInputRetained: retained,
		Retryable:             retryable,
		Billable:              false,
		ScoringAffected:       false,
		RetryAction:           action,
	}
}

// UploadRejectedError is a pre-storage validation rejection safe to show to users.
type UploadRejectedError struct {
	Reason  RejectionReason
	Message string
	Impact  UploadImpact
}

// NewUploadRejectedError constructs a non-billable rejection with no retained input.
func NewUploadRejectedError(reason RejectionReason, message string) *UploadRejectedError {
	return &UploadRejectedError{Reason: reason, Message: message, Impact: safeImpact(false, false, "")}
}

func (e *UploadRejectedError) Error() string { return string(e.Reason) + ": " + e.Message }
