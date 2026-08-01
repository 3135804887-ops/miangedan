// Package statemachine 提供项目状态机的确定性引擎（TASK-017）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-017；docs/domain/INTERVIEW-STATE-MACHINE.md 第 5 节；
// ADR-0001（LLM 无权直接改变业务状态；状态迁移只由确定性事件驱动）。
package statemachine

import (
	"errors"
	"fmt"
)

// Status 为项目状态（与 INTERVIEW-STATE-MACHINE.md 5.1 逐条一致）。
type Status string

// 状态全集（15 个，与 openapi ProjectStatus 一致）。
const (
	StatusDraft                Status = "DRAFT"
	StatusParsing              Status = "PARSING"
	StatusMaterialReview       Status = "MATERIAL_REVIEW"
	StatusParseFailed          Status = "PARSE_FAILED"
	StatusPlanGenerating       Status = "PLAN_GENERATING"
	StatusPlanReview           Status = "PLAN_REVIEW"
	StatusPlanFailed           Status = "PLAN_FAILED"
	StatusReady                Status = "READY"
	StatusInSession            Status = "IN_SESSION"
	StatusScoring              Status = "SCORING"
	StatusRoundPassed          Status = "ROUND_PASSED"
	StatusRoundFailed          Status = "ROUND_FAILED"
	StatusPracticing           Status = "PRACTICING"
	StatusEvaluationIncomplete Status = "EVALUATION_INCOMPLETE"
	StatusCompleted            Status = "COMPLETED"
)

// Event 为项目级确定性事件（INTERVIEW-STATE-MACHINE.md 5.2 事件名）。
type Event string

// 事件全集（22 个）。
const (
	EventMaterialsSubmitted   Event = "materials.submitted"
	EventParseSucceeded       Event = "parse.succeeded"
	EventParseFailed          Event = "parse.failed"
	EventParseRetryRequested  Event = "parse.retry_requested"
	EventMaterialsConfirmed   Event = "materials.confirmed"
	EventPlanGenerated        Event = "plan.generated"
	EventPlanFailed           Event = "plan.failed"
	EventPlanRetryRequested   Event = "plan.retry_requested"
	EventPlanConfirmed        Event = "plan.confirmed"
	EventRoundStarted         Event = "round.started"
	EventRoundEnded           Event = "round.ended"
	EventSessionUserExited    Event = "session.user_exited"
	EventSessionUnrecoverable Event = "session.unrecoverable"
	EventScoringPassed        Event = "scoring.passed"
	EventScoringFailedGate    Event = "scoring.failed_gate"
	EventScoringIncomplete    Event = "scoring.incomplete"
	EventHandoffReady         Event = "handoff.ready"
	// #nosec G101 -- 状态机事件名（非凭证），命名来自 INTERVIEW-STATE-MACHINE.md 5.2 契约
	EventAllRoundsPassed    Event = "project.all_rounds_passed"
	EventPracticeStarted    Event = "practice.started"
	EventPracticeEnded      Event = "practice.ended"
	EventRetryStarted       Event = "retry.started"
	EventProjectEndedByUser Event = "project.ended_by_user"
)

// 引擎错误。
var (
	ErrTerminal          = errors.New("状态机已到达终态，不再接受迁移")
	ErrInvalidTransition = errors.New("当前状态不接受该事件")
	ErrEndedByUser       = errors.New("评估未完成且用户已结束整场（终态分支，生成部分报告）")
)

// transitions 为 INTERVIEW-STATE-MACHINE.md 5.2 迁移表的纯映射。
var transitions = map[Status]map[Event]Status{
	StatusDraft:                {EventMaterialsSubmitted: StatusParsing},
	StatusParsing:              {EventParseSucceeded: StatusMaterialReview, EventParseFailed: StatusParseFailed},
	StatusParseFailed:          {EventParseRetryRequested: StatusParsing},
	StatusMaterialReview:       {EventMaterialsConfirmed: StatusPlanGenerating},
	StatusPlanGenerating:       {EventPlanGenerated: StatusPlanReview, EventPlanFailed: StatusPlanFailed},
	StatusPlanFailed:           {EventPlanRetryRequested: StatusPlanGenerating},
	StatusPlanReview:           {EventPlanConfirmed: StatusReady},
	StatusReady:                {EventRoundStarted: StatusInSession},
	StatusInSession:            {EventRoundEnded: StatusScoring, EventSessionUserExited: StatusEvaluationIncomplete, EventSessionUnrecoverable: StatusEvaluationIncomplete},
	StatusScoring:              {EventScoringPassed: StatusRoundPassed, EventScoringFailedGate: StatusRoundFailed, EventScoringIncomplete: StatusEvaluationIncomplete},
	StatusRoundPassed:          {EventHandoffReady: StatusReady, EventAllRoundsPassed: StatusCompleted},
	StatusRoundFailed:          {EventPracticeStarted: StatusPracticing, EventRetryStarted: StatusReady},
	StatusPracticing:           {EventPracticeEnded: StatusRoundFailed},
	StatusEvaluationIncomplete: {EventRetryStarted: StatusReady, EventProjectEndedByUser: StatusEvaluationIncomplete},
	StatusCompleted:            {},
}

// ProjectState 为项目状态机的确定性快照（Temporal 重放安全：无随机、无系统时钟依赖）。
type ProjectState struct {
	Status       Status
	CurrentRound int
	TotalRounds  int
	PassedRounds int
	EndedByUser  bool
}

// NewProject 创建 DRAFT 项目状态；TotalRounds 必须 ≥1（由计划确认时写入）。
func NewProject(totalRounds int) (ProjectState, error) {
	if totalRounds < 1 || totalRounds > 5 {
		return ProjectState{}, fmt.Errorf("轮次数必须为 1-5（FR-009），实际 %d", totalRounds)
	}
	return ProjectState{Status: StatusDraft, TotalRounds: totalRounds}, nil
}

// Terminal 判断是否到达终态（COMPLETED 或用户结束整场的评估未完成分支）。
func (s ProjectState) Terminal() bool {
	return s.Status == StatusCompleted || s.EndedByUser
}

// Apply 应用事件并迁移状态；非法迁移返回 ErrInvalidTransition；
// 用户结束整场（project.ended_by_user）仅在 EVALUATION_INCOMPLETE 接受，状态保持并置 EndedByUser。
func (s *ProjectState) Apply(event Event) error {
	if s.Terminal() {
		return ErrTerminal
	}
	if event == EventProjectEndedByUser {
		if s.Status != StatusEvaluationIncomplete {
			return fmt.Errorf("%w: %s 仅允许在 EVALUATION_INCOMPLETE 应用", ErrInvalidTransition, event)
		}
		s.EndedByUser = true
		return nil
	}
	next, ok := transitions[s.Status][event]
	if !ok {
		return fmt.Errorf("%w: %s 不能从 %s 迁移", ErrInvalidTransition, event, s.Status)
	}
	s.Status = next
	switch event {
	case EventScoringPassed:
		s.PassedRounds++
	case EventRetryStarted:
		s.PassedRounds = 0 // 正式重试重新计数当前轮
	case EventRoundStarted:
		// CurrentRound 由工作流按信号轮次序号写入（见 StartRound）
	}
	return nil
}

// StartRound 记录当前轮序号（round.started 前置：状态必须已迁移为 IN_SESSION）。
func (s *ProjectState) StartRound(sequence int) error {
	if s.Status != StatusInSession {
		return fmt.Errorf("%w: 只有 IN_SESSION 可记录当前轮", ErrInvalidTransition)
	}
	if sequence < 1 || sequence > s.TotalRounds {
		return fmt.Errorf("轮次序号 %d 超出计划范围 1-%d", sequence, s.TotalRounds)
	}
	s.CurrentRound = sequence
	return nil
}

// AllRoundsPassed 判断已通过轮次是否达到计划轮数（工作流据此自动触发 project.all_rounds_passed）。
func (s ProjectState) AllRoundsPassed() bool {
	return s.Status == StatusRoundPassed && s.PassedRounds >= s.TotalRounds
}
