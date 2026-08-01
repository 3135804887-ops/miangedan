package statemachine

import (
	"errors"
	"testing"
)

// 正常路径：5.2 迁移表逐行验证（22 条）。
func TestTransitionsTable(t *testing.T) {
	table := []struct {
		from  Status
		event Event
		to    Status
	}{
		{StatusDraft, EventMaterialsSubmitted, StatusParsing},
		{StatusParsing, EventParseSucceeded, StatusMaterialReview},
		{StatusParsing, EventParseFailed, StatusParseFailed},
		{StatusParseFailed, EventParseRetryRequested, StatusParsing},
		{StatusMaterialReview, EventMaterialsConfirmed, StatusPlanGenerating},
		{StatusPlanGenerating, EventPlanGenerated, StatusPlanReview},
		{StatusPlanGenerating, EventPlanFailed, StatusPlanFailed},
		{StatusPlanFailed, EventPlanRetryRequested, StatusPlanGenerating},
		{StatusPlanReview, EventPlanConfirmed, StatusReady},
		{StatusReady, EventRoundStarted, StatusInSession},
		{StatusInSession, EventRoundEnded, StatusScoring},
		{StatusInSession, EventSessionUserExited, StatusEvaluationIncomplete},
		{StatusInSession, EventSessionUnrecoverable, StatusEvaluationIncomplete},
		{StatusScoring, EventScoringPassed, StatusRoundPassed},
		{StatusScoring, EventScoringFailedGate, StatusRoundFailed},
		{StatusScoring, EventScoringIncomplete, StatusEvaluationIncomplete},
		{StatusRoundPassed, EventHandoffReady, StatusReady},
		{StatusRoundPassed, EventAllRoundsPassed, StatusCompleted},
		{StatusRoundFailed, EventPracticeStarted, StatusPracticing},
		{StatusPracticing, EventPracticeEnded, StatusRoundFailed},
		{StatusRoundFailed, EventRetryStarted, StatusReady},
		{StatusEvaluationIncomplete, EventRetryStarted, StatusReady},
	}
	for _, row := range table {
		s, err := NewProject(3)
		if err != nil {
			t.Fatal(err)
		}
		s.Status = row.from
		if err := s.Apply(row.event); err != nil {
			t.Fatalf("%s --%s--> %s 应成功: %v", row.from, row.event, row.to, err)
		}
		if s.Status != row.to {
			t.Fatalf("%s --%s--> 应为 %s，实际 %s", row.from, row.event, row.to, s.Status)
		}
	}
}

// 异常路径：非法迁移必须拒绝（含终态后任何事件）。
func TestInvalidTransitionsRejected(t *testing.T) {
	cases := []struct {
		from  Status
		event Event
		want  error
	}{
		{StatusDraft, EventPlanConfirmed, ErrInvalidTransition},
		{StatusReady, EventParseSucceeded, ErrInvalidTransition},
		{StatusCompleted, EventRetryStarted, ErrTerminal},
		{StatusPracticing, EventRoundStarted, ErrInvalidTransition},
	}
	for _, c := range cases {
		s, err := NewProject(3)
		if err != nil {
			t.Fatal(err)
		}
		s.Status = c.from
		if err := s.Apply(c.event); !errors.Is(err, c.want) {
			t.Fatalf("%s --%s--> 必须返回 %v，实际 %v", c.from, c.event, c.want, err)
		}
	}
}

// 终态分支：EVALUATION_INCOMPLETE + project.ended_by_user 保持状态并置 EndedByUser；之后拒绝任何事件。
func TestEndedByUserTerminalBranch(t *testing.T) {
	s, err := NewProject(3)
	if err != nil {
		t.Fatal(err)
	}
	s.Status = StatusEvaluationIncomplete
	if err := s.Apply(EventProjectEndedByUser); err != nil {
		t.Fatalf("终态分支应接受: %v", err)
	}
	if !s.EndedByUser || s.Status != StatusEvaluationIncomplete || !s.Terminal() {
		t.Fatalf("应保持 EVALUATION_INCOMPLETE 且 EndedByUser：%+v", s)
	}
	if err := s.Apply(EventRetryStarted); !errors.Is(err, ErrTerminal) {
		t.Fatalf("用户结束后任何事件必须拒绝，实际 %v", err)
	}
}

// 正常路径：完整旅程 DRAFT → COMPLETED，含轮次推进与通过计数。
func TestFullJourney(t *testing.T) {
	s, err := NewProject(2)
	if err != nil {
		t.Fatal(err)
	}
	events := []Event{
		EventMaterialsSubmitted, EventParseSucceeded, EventMaterialsConfirmed,
		EventPlanGenerated, EventPlanConfirmed,
		EventRoundStarted, EventRoundEnded, EventScoringPassed, EventHandoffReady,
		EventRoundStarted, EventRoundEnded, EventScoringPassed,
	}
	round := 0
	for _, e := range events {
		if err := s.Apply(e); err != nil {
			t.Fatalf("事件 %s 应用失败: %v", e, err)
		}
		if e == EventRoundStarted {
			round++
			if err := s.StartRound(round); err != nil {
				t.Fatalf("第 %d 轮 StartRound 失败: %v", round, err)
			}
		}
	}
	if s.PassedRounds != 2 {
		t.Fatalf("已通过轮数应为 2：%+v", s)
	}
	if !s.AllRoundsPassed() {
		t.Fatal("两轮通过后应满足 AllRoundsPassed")
	}
	if err := s.Apply(EventAllRoundsPassed); err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusCompleted || !s.Terminal() {
		t.Fatalf("应到达 COMPLETED：%+v", s)
	}
}

// 异常路径：非法轮次序号必须拒绝。
func TestStartRoundRejected(t *testing.T) {
	s, err := NewProject(3)
	if err != nil {
		t.Fatal(err)
	}
	s.Status = StatusInSession
	if err := s.StartRound(4); err == nil {
		t.Fatal("超出计划轮数必须拒绝")
	}
	s.Status = StatusReady
	if err := s.StartRound(1); err == nil {
		t.Fatal("非 IN_SESSION 不得记录当前轮")
	}
}
