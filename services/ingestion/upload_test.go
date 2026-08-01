package ingestion

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

type countingSandbox struct {
	runner SandboxRunner
	calls  atomic.Int32
}

func (s *countingSandbox) Attestation() SandboxAttestation { return s.runner.Attestation() }

func (s *countingSandbox) Scan(ctx context.Context, request ScanRequest) (ScanReport, error) {
	s.calls.Add(1)
	return s.runner.Scan(ctx, request)
}

type sequenceSandbox struct {
	mu      sync.Mutex
	results []error
	scanner Scanner
	calls   int
}

func (s *sequenceSandbox) Attestation() SandboxAttestation { return compliantAttestation() }

func (s *sequenceSandbox) Scan(ctx context.Context, request ScanRequest) (ScanReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if len(s.results) > 0 {
		err := s.results[0]
		s.results = s.results[1:]
		if err != nil {
			return ScanReport{}, err
		}
	}
	return s.scanner.Scan(ctx, request)
}

func compliantAttestation() SandboxAttestation {
	return SandboxAttestation{
		NetworkDisabled:        true,
		Ephemeral:              true,
		ReadOnlyRootFilesystem: true,
		CredentialsMounted:     false,
	}
}

func validPDF() []byte { return []byte("%PDF-1.7\n1 0 obj<<>>endobj\n%%EOF") }

func validDOC() []byte {
	return append(append([]byte{}, oleHeader...), []byte("synthetic word document")...)
}

func docx(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var output bytes.Buffer
	w := zip.NewWriter(&output)
	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func validDOCX(t *testing.T) []byte {
	return docx(t, map[string][]byte{
		"[Content_Types].xml": []byte("<Types/>"),
		"word/document.xml":   []byte("<document>synthetic resume</document>"),
	})
}

func newTestService(t *testing.T, sandbox SandboxRunner) (*Service, *MemoryRepository, *MemoryUploadStore) {
	t.Helper()
	repo := NewMemoryRepository()
	store := NewMemoryUploadStore()
	var ids atomic.Int32
	service, err := NewService(repo, store, sandbox, func() string {
		return fmt.Sprintf("00000000-0000-7000-8000-%012d", ids.Add(1))
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repo, store
}

func request(filename string, content []byte, idem string) UploadRequest {
	return UploadRequest{
		UserID:         "00000000-0000-7000-8000-000000000001",
		DataRegion:     "cn",
		IdempotencyKey: idem,
		Filename:       filename,
		DeclaredSize:   int64(len(content)),
		Content:        content,
	}
}

func TestUploadAcceptsSupportedFilesInRegionalUploadsBucket(t *testing.T) {
	runner := &countingSandbox{runner: AttestedSandbox{
		Policy:  compliantAttestation(),
		Scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	}}
	service, _, store := newTestService(t, runner)
	cases := []struct {
		name, filename, media string
		content               []byte
	}{
		{"pdf", "resume.pdf", "application/pdf", validPDF()},
		{"doc", "resume.doc", "application/msword", validDOC()},
		{"docx", "resume.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", validDOCX(t)},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upload, err := service.Upload(context.Background(), request(tc.filename, tc.content, fmt.Sprintf("idem-valid-%02d", i)))
			if err != nil {
				t.Fatal(err)
			}
			if upload.Status != StatusAccepted || upload.DetectedMediaType != tc.media || upload.ObjectRef == nil {
				t.Fatalf("unexpected accepted upload: %#v", upload)
			}
			if upload.ObjectRef.Bucket != "cn-uploads" || !store.Has(*upload.ObjectRef) {
				t.Fatalf("accepted object must remain only in regional uploads bucket: %#v", upload.ObjectRef)
			}
			if upload.Impact.Billable || upload.Impact.ScoringAffected {
				t.Fatal("upload scanning must not bill or affect scoring")
			}
		})
	}
}

func TestMaliciousFileMatrixRejectedWithSpecificReasons(t *testing.T) {
	macroDOC := append(validDOC(), []byte("_VBA_PROJECT")...)
	encryptedPDF := []byte("%PDF-1.7\n/Encrypt 2 0 R\n%%EOF")
	spoofed := validDOC()
	corrupted := []byte("%PDF-1.7\nmissing eof")
	virus := append(validPDF(), []byte("SYNTHETIC-MALWARE-SIGNATURE")...)
	macroDOCX := docx(t, map[string][]byte{
		"[Content_Types].xml": []byte("<Types/>"),
		"word/document.xml":   []byte("<document/>"),
		"word/vbaProject.bin": []byte("synthetic macro"),
	})
	bombDOCX := docx(t, map[string][]byte{
		"[Content_Types].xml": []byte("<Types/>"),
		"word/document.xml":   bytes.Repeat([]byte("A"), 2*1024*1024),
	})
	cases := []struct {
		name, filename string
		content        []byte
		reason         RejectionReason
	}{
		{"virus", "resume.pdf", virus, ReasonVirusDetected},
		{"macro-doc", "resume.doc", macroDOC, ReasonMacrosDetected},
		{"macro-docx", "resume.docx", macroDOCX, ReasonMacrosDetected},
		{"archive-bomb", "resume.docx", bombDOCX, ReasonArchiveBombDetected},
		{"type-spoofed", "resume.pdf", spoofed, ReasonTypeSpoofed},
		{"corrupted", "resume.pdf", corrupted, ReasonCorrupted},
		{"encrypted", "resume.pdf", encryptedPDF, ReasonEncrypted},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, _, store := newTestService(t, AttestedSandbox{
				Policy:  compliantAttestation(),
				Scanner: Scanner{Malware: SyntheticSignatureDetector{}},
			})
			upload, err := service.Upload(context.Background(), request(tc.filename, tc.content, fmt.Sprintf("idem-matrix-%02d", i)))
			if err != nil {
				t.Fatal(err)
			}
			if upload.Status != StatusRejected || upload.RejectionReason != tc.reason {
				t.Fatalf("expected %s, got %#v", tc.reason, upload)
			}
			if upload.Message == "" || upload.ObjectRef != nil || upload.Impact.OriginalInputRetained || upload.Impact.Retryable {
				t.Fatalf("security rejection must be specific, deleted, and non-retryable: %#v", upload)
			}
			if store.DeleteCount != 1 {
				t.Fatalf("rejected quarantine object must be deleted once, got %d", store.DeleteCount)
			}
		})
	}
}

func TestOversizedAndUnsupportedRejectedBeforeObjectStorage(t *testing.T) {
	service, _, store := newTestService(t, AttestedSandbox{
		Policy:  compliantAttestation(),
		Scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	})
	cases := []struct {
		name string
		req  UploadRequest
		want RejectionReason
	}{
		{"oversized", request("resume.pdf", bytes.Repeat([]byte("S"), int(MaxResumeBytes)+1), "idem-over-limit"), ReasonOversized},
		{"unsupported", request("resume.exe", []byte("MZ synthetic"), "idem-unsupported"), ReasonUnsupportedType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Upload(context.Background(), tc.req)
			var rejected *UploadRejectedError
			if !errors.As(err, &rejected) || rejected.Reason != tc.want {
				t.Fatalf("expected %s rejection, got %v", tc.want, err)
			}
		})
	}
	if store.PutCount != 0 {
		t.Fatalf("pre-validation rejection must not write any bucket, got %d puts", store.PutCount)
	}
}

func TestReadAllLimitedRejectsActualOversize(t *testing.T) {
	content := bytes.Repeat([]byte("S"), int(MaxResumeBytes)+1)
	_, err := ReadAllLimited(bytes.NewReader(content))
	var rejected *UploadRejectedError
	if !errors.As(err, &rejected) || rejected.Reason != ReasonOversized {
		t.Fatalf("stream limit must reject actual oversize, got %v", err)
	}
}

func TestScannerUnavailableRetainsQuarantineForRetry(t *testing.T) {
	runner := &sequenceSandbox{
		results: []error{ErrScannerUnavailable},
		scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	}
	service, _, store := newTestService(t, runner)
	upload, err := service.Upload(context.Background(), request("resume.pdf", validPDF(), "idem-scanner-down"))
	if err != nil {
		t.Fatal(err)
	}
	if upload.Status != StatusRetryableFailed || upload.ObjectRef == nil || !upload.Impact.Retryable || !store.Has(*upload.ObjectRef) {
		t.Fatalf("scanner outage must retain quarantined input for retry: %#v", upload)
	}
}

func TestSandboxPolicyFailsClosedBeforeUpload(t *testing.T) {
	_, err := NewService(NewMemoryRepository(), NewMemoryUploadStore(), AttestedSandbox{
		Policy:  SandboxAttestation{NetworkDisabled: false, Ephemeral: true, ReadOnlyRootFilesystem: true},
		Scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	}, func() string { return "id" })
	if err == nil {
		t.Fatal("network-enabled sandbox must be rejected before processing")
	}
}

func TestScanTimeoutRetainsOriginalAndRetryIsIdempotent(t *testing.T) {
	runner := &sequenceSandbox{
		results: []error{ErrScanTimeout, nil},
		scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	}
	service, _, store := newTestService(t, runner)
	content := validPDF()
	upload, err := service.Upload(context.Background(), request("resume.pdf", content, "idem-timeout-first"))
	if err != nil {
		t.Fatal(err)
	}
	if upload.Status != StatusRetryableFailed || upload.ObjectRef == nil || !upload.Impact.OriginalInputRetained || !upload.Impact.Retryable {
		t.Fatalf("timeout must retain original for retry: %#v", upload)
	}
	if !store.Has(*upload.ObjectRef) {
		t.Fatal("retained quarantine object missing")
	}
	retried, err := service.Retry(context.Background(), upload.UploadID, "idem-timeout-retry")
	if err != nil || retried.Status != StatusAccepted {
		t.Fatalf("retry should accept original: %#v, %v", retried, err)
	}
	duplicate, err := service.Retry(context.Background(), upload.UploadID, "idem-timeout-retry")
	if err != nil || duplicate.Status != StatusAccepted {
		t.Fatalf("duplicate retry must return first result: %#v, %v", duplicate, err)
	}
	if runner.calls != 2 || store.PutCount != 1 || store.PromoteCount != 1 {
		t.Fatalf("retry side effects duplicated: scans=%d puts=%d promotes=%d", runner.calls, store.PutCount, store.PromoteCount)
	}
}

func TestUploadIdempotencyAndConcurrentDuplicates(t *testing.T) {
	runner := &countingSandbox{runner: AttestedSandbox{
		Policy:  compliantAttestation(),
		Scanner: Scanner{Malware: SyntheticSignatureDetector{}},
	}}
	service, _, store := newTestService(t, runner)
	req := request("resume.pdf", validPDF(), "idem-concurrent")
	const workers = 16
	results := make(chan Upload, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			upload, err := service.Upload(context.Background(), req)
			results <- upload
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	var uploadID string
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for upload := range results {
		if uploadID == "" {
			uploadID = upload.UploadID
		}
		if upload.UploadID != uploadID {
			t.Fatalf("duplicate idempotency key returned multiple uploads: %s != %s", upload.UploadID, uploadID)
		}
	}
	if store.PutCount != 1 || runner.calls.Load() != 1 {
		t.Fatalf("concurrent duplicate side effects: puts=%d scans=%d", store.PutCount, runner.calls.Load())
	}
	changed := req
	changed.Content = []byte("%PDF-1.7\nchanged\n%%EOF")
	changed.DeclaredSize = int64(len(changed.Content))
	if _, err := service.Upload(context.Background(), changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("same key with different payload must conflict, got %v", err)
	}
}
