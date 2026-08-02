// TASK-024 岗位工具（FR-019）。
// 追踪：docs/api/realtime-events.md 7.5（tool.activated / tool.event / tool.snapshot）；
// SCREEN-SPEC SCR-08（未配置工具禁止临时加载）；NFR-005（工具事件入证据账本）。
package room

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"miangedan/services/project"
)

// ToolKey 为岗位工具类型（与 project.ToolTypes 一致，禁止自创）。
type ToolKey string

// 岗位工具枚举（PRD FR-019）。
const (
	ToolCodeEditor    ToolKey = "code_editor"
	ToolWhiteboard    ToolKey = "whiteboard"
	ToolCaseMaterials ToolKey = "case_materials"
	ToolPortfolio     ToolKey = "portfolio"
)

// ToolEventType 为工具事件类型（realtime-events 7.5）。
type ToolEventType string

// 工具事件枚举。
const (
	ToolEventEdit     ToolEventType = "edit"
	ToolEventRun      ToolEventType = "run"
	ToolEventAnnotate ToolEventType = "annotate"
	ToolEventSubmit   ToolEventType = "submit"
)

// ToolEvent 为单条工具事件（入证据账本；content_ref 为对象存储引用，非内联大对象）。
type ToolEvent struct {
	SessionID   string
	DataRegion  string
	ToolKey     ToolKey
	ToolEventID string
	EventType   ToolEventType
	ContentRef  string
	CreatedAt   time.Time
}

// ActivateToolInput 为激活工具入参。
type ActivateToolInput struct {
	ToolKey      ToolKey
	PreconfigRef string
}

// ToolActivation 为工具激活结果（tool.activated；不产生证据事件）。
type ToolActivation struct {
	SessionID    string
	DataRegion   string
	ToolKey      ToolKey
	PreconfigRef string
	ActivatedAt  time.Time
}

// 工具相关错误。
var (
	ErrToolInvalid       = errors.New("invalid tool")
	ErrToolNotConfigured = errors.New("tool not configured")
)

var toolKeys = map[ToolKey]bool{
	ToolCodeEditor: true, ToolWhiteboard: true,
	ToolCaseMaterials: true, ToolPortfolio: true,
}

var toolEventTypes = map[ToolEventType]bool{
	ToolEventEdit: true, ToolEventRun: true,
	ToolEventAnnotate: true, ToolEventSubmit: true,
}

// roundTools 读取计划中该轮的已配置工具（FR-019：未配置工具禁止临时加载）。
func (s *Service) roundTools(ctx context.Context, actor project.Actor, sess Session) (map[ToolKey]bool, error) {
	plan, err := s.projects.GetPlan(ctx, actor, sess.ProjectID)
	if err != nil {
		return nil, mapProjectErr(err)
	}
	out := make(map[ToolKey]bool)
	for _, r := range plan.Rounds {
		if r.Sequence == sess.RoundSequence {
			for _, t := range r.Tools {
				out[ToolKey(t)] = true
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("%w: 计划中不存在轮次 %d", ErrStateConflict, sess.RoundSequence)
}

// ActivateTool 激活岗位工具（tool.activated）：仅允许计划中已配置工具。
func (s *Service) ActivateTool(ctx context.Context, actor project.Actor, sessionID string, in ActivateToolInput) (ToolActivation, error) {
	if err := s.validateActor(actor); err != nil {
		return ToolActivation{}, err
	}
	if !toolKeys[in.ToolKey] {
		return ToolActivation{}, fmt.Errorf("%w: 未知工具类型 %q", ErrToolInvalid, in.ToolKey)
	}
	sess, err := s.ownSession(actor, sessionID)
	if err != nil {
		return ToolActivation{}, err
	}
	if sess.RoomStatus == StatusEnded {
		return ToolActivation{}, fmt.Errorf("%w: 会话已结束", ErrStateConflict)
	}
	configured, err := s.roundTools(ctx, actor, sess)
	if err != nil {
		return ToolActivation{}, err
	}
	if !configured[in.ToolKey] {
		return ToolActivation{}, fmt.Errorf("%w: 该轮未配置 %s（正式房间不临时加载工具）", ErrToolNotConfigured, in.ToolKey)
	}
	return ToolActivation{
		SessionID:    sessionID,
		DataRegion:   actor.DataRegion,
		ToolKey:      in.ToolKey,
		PreconfigRef: in.PreconfigRef,
		ActivatedAt:  s.now(),
	}, nil
}

// RecordToolEvent 记录工具事件（tool.event；幂等键 tool_event_id；入证据账本）。
func (s *Service) RecordToolEvent(ctx context.Context, actor project.Actor, sessionID string, in ToolEvent, idemKey string) (ToolEvent, error) {
	if err := s.validateActor(actor); err != nil {
		return ToolEvent{}, err
	}
	if !toolKeys[in.ToolKey] {
		return ToolEvent{}, fmt.Errorf("%w: 未知工具类型", ErrToolInvalid)
	}
	if !toolEventTypes[in.EventType] {
		return ToolEvent{}, fmt.Errorf("%w: 未知事件类型", ErrToolInvalid)
	}
	if strings.TrimSpace(in.ToolEventID) == "" {
		return ToolEvent{}, fmt.Errorf("%w: tool_event_id 必填", ErrToolInvalid)
	}
	if in.EventType != ToolEventSubmit && strings.TrimSpace(in.ContentRef) == "" {
		return ToolEvent{}, fmt.Errorf("%w: content_ref 必填（对象存储引用，非内联大对象）", ErrToolInvalid)
	}
	sess, err := s.ownSession(actor, sessionID)
	if err != nil {
		return ToolEvent{}, err
	}
	if sess.RoomStatus == StatusEnded {
		return ToolEvent{}, fmt.Errorf("%w: 会话已结束", ErrStateConflict)
	}
	configured, err := s.roundTools(ctx, actor, sess)
	if err != nil {
		return ToolEvent{}, err
	}
	if !configured[in.ToolKey] {
		return ToolEvent{}, fmt.Errorf("%w: 未配置工具不可产生事件", ErrToolNotConfigured)
	}
	return idempotent(s, "toolevent|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (ToolEvent, error) {
		ev := in
		ev.SessionID = sessionID
		ev.DataRegion = actor.DataRegion
		ev.CreatedAt = s.now()
		if err := s.store.SaveToolEvent(ev); err != nil {
			return ToolEvent{}, err
		}
		return ev, nil
	})
}

// ListToolEvents 列出会话工具事件（按创建时间升序，供报告与复核引用）。
func (s *Service) ListToolEvents(_ context.Context, actor project.Actor, sessionID string) ([]ToolEvent, error) {
	if err := s.validateActor(actor); err != nil {
		return nil, err
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return nil, err
	}
	items, err := s.store.ListToolEvents(actor.DataRegion, sessionID)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
