// TASK-023 双向字幕与转写修订（FR-018）。
// 追踪：docs/api/realtime-events.md 7.3（caption/asr/transcript.revision/turn.completed）；
// NFR-005（证据持久化边界）；INTERVIEW-STATE-MACHINE（回合冻结）。
package room

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/project"
)

// TranscriptKind 为转写文本类型（realtime-events 7.3）。
type TranscriptKind string

// 转写类型：partial 仅展示不入证据；final 为正式 ASR 文本（原始文本仅诊断）。
const (
	TranscriptPartial TranscriptKind = "partial"
	TranscriptFinal   TranscriptKind = "final"
)

// RevisionState 为修订状态机。
type RevisionState string

// 修订状态：none 未修订；submitted 已提交待冻结；accepted 成为评分证据；rejected 被拒。
const (
	RevisionNone      RevisionState = "none"
	RevisionSubmitted RevisionState = "submitted"
	RevisionAccepted  RevisionState = "accepted"
	RevisionRejected  RevisionState = "rejected"
)

// Transcript 为单条字幕/转写（原始 ASR 与修订双版本，原始仅诊断）。
type Transcript struct {
	SessionID              string
	DataRegion             string
	TurnIndex              int
	UtteranceID            string
	Kind                   TranscriptKind
	Text                   string
	Language               string
	Confidence             float64
	RevisedText            string
	RevisionID             string
	RevisionState          RevisionState
	RevisionRejectedReason string
	Frozen                 bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// TurnState 为回合冻结边界（turn.completed 前可修订，之后拒绝）。
type TurnState struct {
	SessionID  string
	DataRegion string
	TurnIndex  int
	Frozen     bool
	FrozenAt   *time.Time
}

// AppendTranscriptInput 为追加 ASR 文本入参（asr.partial / asr.final）。
type AppendTranscriptInput struct {
	TurnIndex   int
	UtteranceID string
	Kind        TranscriptKind
	Text        string
	Language    string
	Confidence  float64
}

// RevisionInput 为修订提交入参（transcript.revision.submitted）。
type RevisionInput struct {
	RevisionID  string
	UtteranceID string
	TurnIndex   int
	RevisedText string
}

// FreezeTurnResult 为回合冻结结果。
type FreezeTurnResult struct {
	SessionID    string
	TurnIndex    int
	FrozenAt     time.Time
	FinalCount   int
	RevisedCount int
}

// 字幕相关错误。
var (
	ErrTranscriptInvalid    = errors.New("invalid transcript")
	ErrRevisionWindowClosed = errors.New("revision window closed")
	ErrTurnAlreadyFrozen    = errors.New("turn already frozen")
	ErrTurnNotFrozen        = errors.New("turn not frozen")
)

func (s *Service) ownSession(actor project.Actor, sessionID string) (Session, error) {
	sess, err := s.store.GetSession(actor.DataRegion, sessionID)
	if err != nil {
		return Session{}, err
	}
	if sess.UserID != actor.UserID {
		return Session{}, ErrNotFound
	}
	return sess, nil
}

// AppendTranscript 追加 ASR 临时/最终文本（realtime-events 7.3）。
// partial 仅覆盖展示文本；final 为正式转写，冻结后拒绝追加。
func (s *Service) AppendTranscript(_ context.Context, actor project.Actor, sessionID string, in AppendTranscriptInput) (Transcript, error) {
	if err := s.validateActor(actor); err != nil {
		return Transcript{}, err
	}
	if in.Kind != TranscriptPartial && in.Kind != TranscriptFinal {
		return Transcript{}, fmt.Errorf("%w: kind 必须为 partial | final", ErrTranscriptInvalid)
	}
	if strings.TrimSpace(in.UtteranceID) == "" {
		return Transcript{}, fmt.Errorf("%w: utterance_id 必填", ErrTranscriptInvalid)
	}
	if strings.TrimSpace(in.Text) == "" {
		return Transcript{}, fmt.Errorf("%w: text 必填", ErrTranscriptInvalid)
	}
	if in.Kind == TranscriptFinal && strings.TrimSpace(in.Language) == "" {
		return Transcript{}, fmt.Errorf("%w: final 转写必须携带语言", ErrTranscriptInvalid)
	}
	if in.Confidence < 0 || in.Confidence > 1 {
		return Transcript{}, fmt.Errorf("%w: confidence 必须位于 [0,1]", ErrTranscriptInvalid)
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return Transcript{}, err
	}
	now := s.now()
	existing, err := s.store.GetTranscript(actor.DataRegion, sessionID, in.UtteranceID)
	exists := err == nil
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Transcript{}, err
	}
	if exists && existing.Frozen && existing.Kind == TranscriptFinal {
		return Transcript{}, fmt.Errorf("%w: 回合已冻结，禁止追加", ErrRevisionWindowClosed)
	}
	t := Transcript{
		SessionID:     sessionID,
		DataRegion:    actor.DataRegion,
		TurnIndex:     in.TurnIndex,
		UtteranceID:   in.UtteranceID,
		Kind:          in.Kind,
		Text:          in.Text,
		Language:      in.Language,
		Confidence:    in.Confidence,
		RevisionState: RevisionNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if exists {
		// 保留已有修订与冻结标记，仅更新展示/最终文本。
		t.RevisionID = existing.RevisionID
		t.RevisionState = existing.RevisionState
		t.RevisionRejectedReason = existing.RevisionRejectedReason
		t.Frozen = existing.Frozen
		t.CreatedAt = existing.CreatedAt
	}
	if err := s.store.SaveTranscript(t); err != nil {
		return Transcript{}, err
	}
	return t, nil
}

// SubmitRevision 提交转写修订（transcript.revision.submitted）。
// 幂等键 revision_id：重放返回首次结果；回合冻结后一律 rejected(window_closed)。
func (s *Service) SubmitRevision(_ context.Context, actor project.Actor, sessionID string, in RevisionInput, idemKey string) (Transcript, error) {
	if err := s.validateActor(actor); err != nil {
		return Transcript{}, err
	}
	if strings.TrimSpace(in.RevisionID) == "" {
		return Transcript{}, fmt.Errorf("%w: revision_id 必填", ErrTranscriptInvalid)
	}
	if strings.TrimSpace(in.UtteranceID) == "" {
		return Transcript{}, fmt.Errorf("%w: utterance_id 必填", ErrTranscriptInvalid)
	}
	if strings.TrimSpace(in.RevisedText) == "" {
		return Transcript{}, fmt.Errorf("%w: revised_text 必填", ErrTranscriptInvalid)
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return Transcript{}, err
	}
	return idempotent(s, "revision|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Transcript, error) {
		t, err := s.store.GetTranscript(actor.DataRegion, sessionID, in.UtteranceID)
		if err != nil {
			return Transcript{}, err
		}
		now := s.now()
		turn, err := s.store.GetTurn(actor.DataRegion, sessionID, in.TurnIndex)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return Transcript{}, err
		}
		if turn.Frozen || t.Frozen {
			t.RevisionState = RevisionRejected
			t.RevisionRejectedReason = "window_closed"
			t.UpdatedAt = now
			if err := s.store.SaveTranscript(t); err != nil {
				return Transcript{}, err
			}
			return t, fmt.Errorf("%w: 进入下一主问题后转写已冻结", ErrRevisionWindowClosed)
		}
		t.RevisionID = in.RevisionID
		t.RevisionState = RevisionSubmitted
		t.RevisionRejectedReason = ""
		t.RevisedText = in.RevisedText
		t.UpdatedAt = now
		if err := s.store.SaveTranscript(t); err != nil {
			return Transcript{}, err
		}
		return t, nil
	})
}

// FreezeTurn 冻结回合（turn.completed，NFR-005：上一有效回答已持久化后发出）。
// 冻结后：修订一律 rejected(window_closed)；final 转写成为评分证据的原始文本快照。
func (s *Service) FreezeTurn(_ context.Context, actor project.Actor, sessionID string, turnIndex int, idemKey string) (FreezeTurnResult, error) {
	if err := s.validateActor(actor); err != nil {
		return FreezeTurnResult{}, err
	}
	if turnIndex < 1 {
		return FreezeTurnResult{}, fmt.Errorf("%w: turn_index 必须 ≥1", ErrTranscriptInvalid)
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return FreezeTurnResult{}, err
	}
	return idempotent(s, "freeze|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (FreezeTurnResult, error) {
		now := s.now()
		turn, err := s.store.GetTurn(actor.DataRegion, sessionID, turnIndex)
		if err == nil && turn.Frozen {
			return FreezeTurnResult{SessionID: sessionID, TurnIndex: turnIndex, FrozenAt: *turn.FrozenAt}, ErrTurnAlreadyFrozen
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return FreezeTurnResult{}, err
		}
		// 冻结前确认该回合全部 final 转写已入库（NFR-005 顺序保证）。
		items, err := s.store.ListTranscripts(actor.DataRegion, sessionID)
		if err != nil {
			return FreezeTurnResult{}, err
		}
		finalCount, revisedCount := 0, 0
		for i := range items {
			if items[i].TurnIndex != turnIndex || items[i].Kind != TranscriptFinal {
				continue
			}
			items[i].Frozen = true
			items[i].UpdatedAt = now
			if items[i].RevisionState == RevisionSubmitted {
				items[i].RevisionState = RevisionAccepted
				revisedCount++
			}
			if err := s.store.SaveTranscript(items[i]); err != nil {
				return FreezeTurnResult{}, err
			}
			finalCount++
		}
		if err := s.store.SaveTurn(TurnState{
			SessionID:  sessionID,
			DataRegion: actor.DataRegion,
			TurnIndex:  turnIndex,
			Frozen:     true,
			FrozenAt:   &now,
		}); err != nil {
			return FreezeTurnResult{}, err
		}
		return FreezeTurnResult{
			SessionID:    sessionID,
			TurnIndex:    turnIndex,
			FrozenAt:     now,
			FinalCount:   finalCount,
			RevisedCount: revisedCount,
		}, nil
	})
}

// ListTranscripts 列出会话全部字幕与转写（含原始/修订双版本）。
func (s *Service) ListTranscripts(_ context.Context, actor project.Actor, sessionID string) ([]Transcript, error) {
	if err := s.validateActor(actor); err != nil {
		return nil, err
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return nil, err
	}
	return s.store.ListTranscripts(actor.DataRegion, sessionID)
}

// GetTurn 读取回合冻结状态（客户端续传游标用）。
func (s *Service) GetTurn(_ context.Context, actor project.Actor, sessionID string, turnIndex int) (TurnState, error) {
	if err := s.validateActor(actor); err != nil {
		return TurnState{}, err
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return TurnState{}, err
	}
	t, err := s.store.GetTurn(actor.DataRegion, sessionID, turnIndex)
	if err != nil {
		return TurnState{}, err
	}
	return t, nil
}
