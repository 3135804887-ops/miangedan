// Package workflow 提供项目状态机 Temporal 工作流（TASK-017）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-017；ADR-0001；docs/domain/INTERVIEW-STATE-MACHINE.md 第 5 节。
// 确定性要求：工作流内不使用随机数或系统时钟，时间一律取 workflow.Now（重放安全）。
package workflow

import (
	"context"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"miangedan/workflows/statemachine"
)

// TaskQueueInterview 为项目状态机任务队列（workflows/README 七域队列：interview 属 project）。
const TaskQueueInterview = "interview"

// 信号与查询名。
const (
	SignalProjectCommand = "project.command"
	QueryProjectState    = "project.state"
)

// Command 为工作流消费的确定性项目事件（LangGraph/服务经此写入业务状态，ADR-0001）。
type Command struct {
	Event          statemachine.Event
	RoundSequence  int
	Reason         string
	ActorID        string
	IdempotencyKey string
}

// ProjectWorkflowInput 为工作流入参。
type ProjectWorkflowInput struct {
	ProjectID   string
	DataRegion  string
	TotalRounds int
}

// ProjectWorkflowResult 为工作流结果（终态快照）。
type ProjectWorkflowResult struct {
	FinalState statemachine.ProjectState
	EndReason  string
}

// AuditRecord 为追加式状态迁移审计（写入 AccessAudit 的契约；实现由审计服务落地）。
type AuditRecord struct {
	ProjectID      string
	DataRegion     string
	Event          statemachine.Event
	From           statemachine.Status
	To             statemachine.Status
	ActorID        string
	IdempotencyKey string
	At             time.Time
}

// ProjectSnapshot 为持久化快照（实现由数据平台落地）。
type ProjectSnapshot struct {
	ProjectID  string
	DataRegion string
	State      statemachine.ProjectState
	UpdatedAt  time.Time
}

// AuditTransitionActivity 追加式审计活动（当前为契约桩；生产实现随审计服务接入）。
func AuditTransitionActivity(_ context.Context, rec AuditRecord) error {
	if auditRecorder != nil {
		auditRecorder(rec)
	}
	return nil
}

// PersistProjectStateActivity 持久化活动（当前为契约桩；生产实现随数据平台接入）。
func PersistProjectStateActivity(_ context.Context, _ ProjectSnapshot) error {
	return nil
}

// auditRecorder 为测试钩子（仅测试注入；生产活动保持契约桩语义）。
var auditRecorder func(AuditRecord)

// ProjectWorkflow 驱动项目状态机；非法迁移仅告警不失败，幂等键重复事件由上层去重（NFR-006）。
func ProjectWorkflow(ctx workflow.Context, input ProjectWorkflowInput) (ProjectWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	state, err := statemachine.NewProject(input.TotalRounds)
	if err != nil {
		return ProjectWorkflowResult{}, err
	}
	if err := workflow.SetQueryHandler(ctx, QueryProjectState, func() (statemachine.ProjectState, error) {
		return state, nil
	}); err != nil {
		return ProjectWorkflowResult{}, err
	}

	sel := workflow.NewSelector(ctx)
	cmdCh := workflow.GetSignalChannel(ctx, SignalProjectCommand)
	result := ProjectWorkflowResult{}
	ended := false

	auditAndPersist := func(from statemachine.Status, event statemachine.Event, cmd Command) {
		snapshot := ProjectSnapshot{
			ProjectID:  input.ProjectID,
			DataRegion: input.DataRegion,
			State:      state,
			UpdatedAt:  workflow.Now(ctx),
		}
		record := AuditRecord{
			ProjectID:      input.ProjectID,
			DataRegion:     input.DataRegion,
			Event:          event,
			From:           from,
			To:             state.Status,
			ActorID:        cmd.ActorID,
			IdempotencyKey: cmd.IdempotencyKey,
			At:             workflow.Now(ctx),
		}
		actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		})
		if err := workflow.ExecuteActivity(actCtx, AuditTransitionActivity, record).Get(actCtx, nil); err != nil {
			logger.Warn("审计活动失败", "event", event, "error", err.Error())
		}
		if err := workflow.ExecuteActivity(actCtx, PersistProjectStateActivity, snapshot).Get(actCtx, nil); err != nil {
			logger.Warn("持久化活动失败", "event", event, "error", err.Error())
		}
	}

	apply := func(cmd Command) {
		from := state.Status
		if err := state.Apply(cmd.Event); err != nil {
			logger.Warn("状态迁移被拒（保持当前状态）", "event", cmd.Event, "from", from, "error", err.Error())
			return
		}
		if cmd.Event == statemachine.EventRoundStarted {
			if err := state.StartRound(cmd.RoundSequence); err != nil {
				logger.Warn("轮次序号非法（状态保持 IN_SESSION）", "sequence", cmd.RoundSequence, "error", err.Error())
			}
		}
		auditAndPersist(from, cmd.Event, cmd)
		// 自动事件：全部必需轮次通过 → project.all_rounds_passed（5.2 迁移表）。
		if state.AllRoundsPassed() {
			autoFrom := state.Status
			if err := state.Apply(statemachine.EventAllRoundsPassed); err != nil {
				logger.Warn("自动完成迁移失败", "error", err.Error())
			} else {
				auditAndPersist(autoFrom, statemachine.EventAllRoundsPassed, cmd)
			}
		}
		if state.Terminal() {
			ended = true
			result.FinalState = state
			result.EndReason = cmd.Reason
		}
	}

	sel.AddReceive(cmdCh, func(ch workflow.ReceiveChannel, _ bool) {
		var cmd Command
		ch.Receive(ctx, &cmd)
		apply(cmd)
	})

	for !ended {
		sel.Select(ctx)
	}
	return result, nil
}
