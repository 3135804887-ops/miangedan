package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"miangedan/workflows/statemachine"
)

func runJourney(t *testing.T, commands []Command) (statemachine.ProjectState, []AuditRecord) {
	t.Helper()
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	var audits []AuditRecord
	auditRecorder = func(r AuditRecord) {
		audits = append(audits, r)
	}
	defer func() { auditRecorder = nil }()
	env.RegisterActivity(AuditTransitionActivity)
	env.RegisterActivity(PersistProjectStateActivity)
	for i, cmd := range commands {
		cmd := cmd
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SignalProjectCommand, cmd)
		}, time.Duration(i)*time.Millisecond)
	}
	env.ExecuteWorkflow(ProjectWorkflow, ProjectWorkflowInput{ProjectID: "p-1", DataRegion: "cn", TotalRounds: 2})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res ProjectWorkflowResult
	require.NoError(t, env.GetWorkflowResult(&res))
	return res.FinalState, audits
}

func cmd(event statemachine.Event) Command {
	return Command{Event: event, ActorID: "user-001", IdempotencyKey: "k-" + string(event)}
}

// 正常路径：两轮完整旅程 DRAFT → COMPLETED，全部轮次通过触发自动完成。
func TestProjectWorkflowFullJourney(t *testing.T) {
	commands := []Command{
		cmd(statemachine.EventMaterialsSubmitted),
		cmd(statemachine.EventParseSucceeded),
		cmd(statemachine.EventMaterialsConfirmed),
		cmd(statemachine.EventPlanGenerated),
		cmd(statemachine.EventPlanConfirmed),
		{Event: statemachine.EventRoundStarted, RoundSequence: 1, ActorID: "u", IdempotencyKey: "k-r1"},
		cmd(statemachine.EventRoundEnded),
		cmd(statemachine.EventScoringPassed),
		cmd(statemachine.EventHandoffReady),
		{Event: statemachine.EventRoundStarted, RoundSequence: 2, ActorID: "u", IdempotencyKey: "k-r2"},
		cmd(statemachine.EventRoundEnded),
		cmd(statemachine.EventScoringPassed),
	}
	state, audits := runJourney(t, commands)
	require.Equal(t, statemachine.StatusCompleted, state.Status)
	require.True(t, state.Terminal())
	require.GreaterOrEqual(t, len(audits), 12, "每次迁移都应写审计")
}

// 异常路径：解析失败 → 重试恢复；会话不可恢复 → 评估未完成 → 用户结束整场（终态分支）。
func TestProjectWorkflowFailureBranch(t *testing.T) {
	commands := []Command{
		cmd(statemachine.EventMaterialsSubmitted),
		cmd(statemachine.EventParseFailed),
		cmd(statemachine.EventParseRetryRequested),
		cmd(statemachine.EventParseSucceeded),
		cmd(statemachine.EventMaterialsConfirmed),
		cmd(statemachine.EventPlanGenerated),
		cmd(statemachine.EventPlanConfirmed),
		{Event: statemachine.EventRoundStarted, RoundSequence: 1, ActorID: "u", IdempotencyKey: "k-r1"},
		cmd(statemachine.EventSessionUnrecoverable),
		cmd(statemachine.EventProjectEndedByUser),
	}
	state, _ := runJourney(t, commands)
	require.Equal(t, statemachine.StatusEvaluationIncomplete, state.Status)
	require.True(t, state.EndedByUser)
	require.True(t, state.Terminal())
}

// 异常路径：非法迁移被拒但工作流继续；评估未完成 → retry.started 回到 READY。
func TestProjectWorkflowInvalidTransitionAndRetry(t *testing.T) {
	commands := []Command{
		cmd(statemachine.EventPlanConfirmed), // DRAFT 下非法，应被拒并保持 DRAFT
		cmd(statemachine.EventMaterialsSubmitted),
		cmd(statemachine.EventParseSucceeded),
		cmd(statemachine.EventMaterialsConfirmed),
		cmd(statemachine.EventPlanGenerated),
		cmd(statemachine.EventPlanConfirmed),
		{Event: statemachine.EventRoundStarted, RoundSequence: 1, ActorID: "u", IdempotencyKey: "k-r1"},
		cmd(statemachine.EventSessionUserExited),
		cmd(statemachine.EventRetryStarted),
		{Event: statemachine.EventRoundStarted, RoundSequence: 1, ActorID: "u", IdempotencyKey: "k-r1b"},
		cmd(statemachine.EventRoundEnded),
		cmd(statemachine.EventScoringPassed),
		cmd(statemachine.EventHandoffReady),
		{Event: statemachine.EventRoundStarted, RoundSequence: 2, ActorID: "u", IdempotencyKey: "k-r2"},
		cmd(statemachine.EventRoundEnded),
		cmd(statemachine.EventScoringPassed),
	}
	state, _ := runJourney(t, commands)
	require.Equal(t, statemachine.StatusCompleted, state.Status)
	require.True(t, state.Terminal())
}
