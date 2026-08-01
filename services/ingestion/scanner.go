package ingestion

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	maxArchiveEntries       = 2_000
	maxExpandedArchiveBytes = 100 * 1024 * 1024
	maxCompressionRatio     = 100
)

var oleHeader = []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}

// MalwareDetector is the vendor-neutral malware detection boundary.
type MalwareDetector interface {
	Detect(ctx context.Context, content []byte) (detected bool, signature string, err error)
}

// Scanner is provider-neutral: production can attach any hardened malware engine without exposing
// vendor SDK semantics to upload business code.
type Scanner struct {
	Malware MalwareDetector
}

// Scan validates malware, file signatures, document structure, macros, and archive limits.
func (s Scanner) Scan(ctx context.Context, request ScanRequest) (ScanReport, error) {
	if s.Malware == nil {
		return ScanReport{}, ErrScannerUnavailable
	}
	if detected, _, err := s.Malware.Detect(ctx, request.Content); err != nil {
		return ScanReport{}, fmt.Errorf("malware detector: %w", err)
	} else if detected {
		return rejected(ReasonVirusDetected, "文件被安全引擎识别为恶意内容，已拒绝并删除隔离副本。"), nil
	}
	ext := strings.ToLower(filepath.Ext(request.Filename))
	if ext == ".docx" && bytes.HasPrefix(request.Content, oleHeader) && bytes.Contains(request.Content, []byte("EncryptedPackage")) {
		return rejected(ReasonEncrypted, "DOCX 已加密，隔离环境无法安全解析；请解除密码后重新上传。"), nil
	}
	kind := sniffKind(request.Content)
	expected := map[string]FileKind{".pdf": FilePDF, ".doc": FileDOC, ".docx": FileDOCX}[ext]
	if kind == "" {
		return rejected(ReasonCorrupted, "文件结构损坏或无法识别；请重新导出后上传。"), nil
	}
	if kind != expected {
		return rejected(ReasonTypeSpoofed, "文件扩展名与实际内容类型不一致，疑似伪装；已拒绝。"), nil
	}
	switch kind {
	case FilePDF:
		return inspectPDF(request.Content)
	case FileDOC:
		return inspectDOC(request.Content)
	case FileDOCX:
		return inspectDOCX(request.Content)
	default:
		return ScanReport{}, errors.New("unsupported sniffed kind")
	}
}

func sniffKind(content []byte) FileKind {
	switch {
	case bytes.HasPrefix(content, []byte("%PDF-")):
		return FilePDF
	case bytes.HasPrefix(content, oleHeader):
		return FileDOC
	case bytes.HasPrefix(content, []byte("PK\x03\x04")):
		return FileDOCX
	default:
		return ""
	}
}

func inspectPDF(content []byte) (ScanReport, error) {
	if bytes.Contains(content, []byte("/Encrypt")) {
		return rejected(ReasonEncrypted, "PDF 已加密，隔离环境无法安全解析；请解除密码后重新上传。"), nil
	}
	if !bytes.Contains(content, []byte("%%EOF")) {
		return rejected(ReasonCorrupted, "PDF 结构不完整或已损坏；请重新导出后上传。"), nil
	}
	if containsMacroMarker(content) {
		return rejected(ReasonMacrosDetected, "文件包含宏或可执行脚本内容，出于安全原因已拒绝。"), nil
	}
	return ScanReport{Kind: FilePDF}, nil
}

func inspectDOC(content []byte) (ScanReport, error) {
	if bytes.Contains(content, []byte("EncryptedPackage")) || bytes.Contains(content, []byte("EncryptionInfo")) {
		return rejected(ReasonEncrypted, "DOC 已加密，隔离环境无法安全解析；请解除密码后重新上传。"), nil
	}
	if containsMacroMarker(content) {
		return rejected(ReasonMacrosDetected, "DOC 包含宏，出于安全原因已拒绝；请另存为无宏 PDF 或 DOCX。"), nil
	}
	return ScanReport{Kind: FileDOC}, nil
}

func inspectDOCX(content []byte) (ScanReport, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return rejected(ReasonCorrupted, "DOCX 压缩结构损坏；请重新导出后上传。"), nil
	}
	if len(reader.File) == 0 || len(reader.File) > maxArchiveEntries {
		return rejected(ReasonArchiveBombDetected, "DOCX 内部条目数量异常，疑似压缩炸弹；已拒绝。"), nil
	}
	var compressed, expanded uint64
	hasContentTypes := false
	hasDocument := false
	for _, file := range reader.File {
		name := strings.ToLower(filepath.ToSlash(file.Name))
		if name == "[content_types].xml" {
			hasContentTypes = true
		}
		if name == "word/document.xml" {
			hasDocument = true
		}
		if strings.HasSuffix(name, "vbaproject.bin") || strings.Contains(name, "/macros/") {
			return rejected(ReasonMacrosDetected, "DOCX 包含 VBA 宏，出于安全原因已拒绝；请另存为无宏文档。"), nil
		}
		compressed += file.CompressedSize64
		expanded += file.UncompressedSize64
		if expanded > maxExpandedArchiveBytes || file.UncompressedSize64 > maxExpandedArchiveBytes {
			return rejected(ReasonArchiveBombDetected, "DOCX 解压后体积超过安全上限，疑似压缩炸弹；已拒绝。"), nil
		}
		denominator := file.CompressedSize64
		if denominator == 0 {
			denominator = 1
		}
		if file.UncompressedSize64/denominator > maxCompressionRatio {
			return rejected(ReasonArchiveBombDetected, "DOCX 单项压缩比异常，疑似压缩炸弹；已拒绝。"), nil
		}
	}
	if compressed == 0 {
		compressed = 1
	}
	if expanded/compressed > maxCompressionRatio {
		return rejected(ReasonArchiveBombDetected, "DOCX 总压缩比异常，疑似压缩炸弹；已拒绝。"), nil
	}
	if !hasContentTypes || !hasDocument {
		return rejected(ReasonCorrupted, "DOCX 缺少必要文档结构；请重新导出后上传。"), nil
	}
	return ScanReport{Kind: FileDOCX}, nil
}

func containsMacroMarker(content []byte) bool {
	lower := bytes.ToLower(content)
	for _, marker := range [][]byte{[]byte("vbaproject"), []byte("_vba_project"), []byte("wordbasic"), []byte("/javascript")} {
		if bytes.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func rejected(reason RejectionReason, message string) ScanReport {
	return ScanReport{RejectionReason: reason, Message: message}
}

// AttestedSandbox is the narrow integration boundary for a one-shot isolated runtime. The runtime
// must produce the attestation; Service.NewService refuses non-compliant runners before any object write.
type AttestedSandbox struct {
	Policy  SandboxAttestation
	Scanner Scanner
}

// Attestation returns the isolation policy asserted by the sandbox runtime.
func (s AttestedSandbox) Attestation() SandboxAttestation { return s.Policy }

// Scan refuses non-compliant isolation and otherwise runs the provider-neutral scanner.
func (s AttestedSandbox) Scan(ctx context.Context, request ScanRequest) (ScanReport, error) {
	if err := s.Policy.Validate(); err != nil {
		return ScanReport{}, fmt.Errorf("%s: %w", ReasonSandboxPolicyViolated, err)
	}
	return s.Scanner.Scan(ctx, request)
}

// SyntheticSignatureDetector is intentionally limited to synthetic fixtures and deterministic tests.
// Production composition must inject a hardened malware engine through MalwareDetector.
type SyntheticSignatureDetector struct{}

// Detect recognizes only the explicit marker reserved for synthetic fixtures.
func (SyntheticSignatureDetector) Detect(_ context.Context, content []byte) (bool, string, error) {
	const marker = "SYNTHETIC-MALWARE-SIGNATURE"
	return bytes.Contains(content, []byte(marker)), marker, nil
}

// ReadAllLimited is shared by an HTTP adapter: it prevents a lying Content-Length from bypassing 10 MiB.
func ReadAllLimited(reader io.Reader) ([]byte, error) {
	limited := io.LimitReader(reader, MaxResumeBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > MaxResumeBytes {
		return nil, NewUploadRejectedError(ReasonOversized, "文件超过 10 MiB 上限，未保存；请压缩或重新导出后重试。")
	}
	return content, nil
}
