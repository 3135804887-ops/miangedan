// Package evidence 提供追加式证据账本写入管道（TASK-026，NFR-005）。
// 追踪：docs/data/DATA-MODEL.md §5.3（evidence_events）；docs/domain/DOMAIN-MODEL.md §6.11；
// realtime-events 7.3/7.5（问题实际播放、回答、修订、工具事件）；ADR-0004（无 UPDATE/DELETE）。
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/region"
)

// Kind 为证据事件类型（与 turn-evidence schema 的 evidence_type 对齐）。
type Kind string

// 证据类型：问题实际播放内容 / 回答 / 修订 / 工具事件。
const (
	KindQuestionPlayed Kind = "question_played"
	KindAnswer         Kind = "answer"
	KindRevision       Kind = "revision"
	KindToolEvent      Kind = "tool_event"
)

// Entry 为一条证据账本条目（全部字段不可变；无更新/删除路径）。
type Entry struct {
	EvidenceID  string
	SessionID   string
	TurnIndex   int
	ProjectID   string
	RoundSeq    int
	AttemptID   string
	Kind        Kind
	EventID     string
	PayloadJSON json.RawMessage
	ContentHash string
	DataRegion  string
	RecordedAt  time.Time
}

// AppendInput 为追加证据入参（实时事件 → 证据管道）。
type AppendInput struct {
	SessionID   string
	TurnIndex   int
	ProjectID   string
	RoundSeq    int
	AttemptID   string
	Kind        Kind
	EventID     string
	PayloadJSON json.RawMessage
}

// Store 为证据账本存储（生产 PostgreSQL；仅 SELECT/INSERT）。
type Store interface {
	SaveEntry(Entry) error
	GetByEventID(dataRegion, eventID string) (Entry, error)
	ListBySession(dataRegion, sessionID string) ([]Entry, error)
}

// 错误集。
var (
	ErrInvalidEntry  = errors.New("invalid evidence entry")
	ErrHashMismatch  = errors.New("evidence content hash mismatch")
	ErrEntryNotFound = errors.New("evidence entry not found")
)

// HashPayload 计算证据载荷的 SHA-256 摘要（入库前必须一致）。
func HashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Service 为追加式证据写入管道（NFR-005：下一主问题前完成上一有效回答持久化）。
// 设计红线：只暴露 Append/Get/List；不存在 Update/Delete 方法（ADR-0004）。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建证据管道服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidEntry)
	}
	return &Service{store: store, now: time.Now}, nil
}

func (s *Service) validate(in AppendInput, dataRegion string) error {
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	if strings.TrimSpace(in.SessionID) == "" || strings.TrimSpace(in.ProjectID) == "" {
		return fmt.Errorf("%w: session_id 与 project_id 必填", ErrInvalidEntry)
	}
	if in.TurnIndex < 1 || in.RoundSeq < 1 {
		return fmt.Errorf("%w: turn_index 与 round_sequence 必须 ≥1", ErrInvalidEntry)
	}
	switch in.Kind {
	case KindQuestionPlayed, KindAnswer, KindRevision, KindToolEvent:
	default:
		return fmt.Errorf("%w: 未知证据类型 %q", ErrInvalidEntry, in.Kind)
	}
	if strings.TrimSpace(in.EventID) == "" {
		return fmt.Errorf("%w: event_id 必填（幂等去重键）", ErrInvalidEntry)
	}
	if len(in.PayloadJSON) == 0 || !json.Valid(in.PayloadJSON) {
		return fmt.Errorf("%w: payload_json 必须为有效 JSON", ErrInvalidEntry)
	}
	return nil
}

// Append 追加一条证据（幂等：同一 event_id 重放返回首条，不产生副作用）。
func (s *Service) Append(_ context.Context, dataRegion string, in AppendInput) (Entry, error) {
	if err := s.validate(in, dataRegion); err != nil {
		return Entry{}, err
	}
	existing, err := s.store.GetByEventID(dataRegion, in.EventID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrEntryNotFound) {
		return Entry{}, err
	}
	hash := HashPayload(in.PayloadJSON)
	entry := Entry{
		EvidenceID:  newID(),
		SessionID:   in.SessionID,
		TurnIndex:   in.TurnIndex,
		ProjectID:   in.ProjectID,
		RoundSeq:    in.RoundSeq,
		AttemptID:   in.AttemptID,
		Kind:        in.Kind,
		EventID:     in.EventID,
		PayloadJSON: append(json.RawMessage(nil), in.PayloadJSON...),
		ContentHash: hash,
		DataRegion:  dataRegion,
		RecordedAt:  s.now(),
	}
	if err := s.store.SaveEntry(entry); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// AppendVerified 追加并校验调用方提供的 content_hash 与载荷一致（fail-closed）。
func (s *Service) AppendVerified(ctx context.Context, dataRegion string, in AppendInput, expectedHash string) (Entry, error) {
	if strings.TrimSpace(expectedHash) == "" {
		return Entry{}, fmt.Errorf("%w: content_hash 必填", ErrInvalidEntry)
	}
	actual := HashPayload(in.PayloadJSON)
	if actual != expectedHash {
		return Entry{}, fmt.Errorf("%w: 载荷与声明哈希不一致", ErrHashMismatch)
	}
	return s.Append(ctx, dataRegion, in)
}

// GetByEventID 幂等键查询（重放/对账）。
func (s *Service) GetByEventID(_ context.Context, dataRegion, eventID string) (Entry, error) {
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return Entry{}, err
	}
	return s.store.GetByEventID(dataRegion, eventID)
}

// ListBySession 列出会话证据（创建时间升序）。
func (s *Service) ListBySession(_ context.Context, dataRegion, sessionID string) ([]Entry, error) {
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return nil, err
	}
	return s.store.ListBySession(dataRegion, sessionID)
}
