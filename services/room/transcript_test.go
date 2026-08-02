package room

import (
	"context"
	"errors"
	"testing"

	"miangedan/services/project"
)

// TASK-023 正常路径：partial → final → 修订 → 冻结 → 修订成为评分证据。
func TestTranscriptFullLifecycle(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t023-create")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	// 数字人/候选人次第追加临时文本（仅展示）。
	if _, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, AppendTranscriptInput{
		TurnIndex: 1, UtteranceID: "utt-1", Kind: TranscriptPartial, Text: "你好", Language: "zh-CN", Confidence: 0.82,
	}); err != nil {
		t.Fatalf("追加 partial 失败: %v", err)
	}
	// 正式 ASR 最终文本（原始文本，仅诊断）。
	final, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, AppendTranscriptInput{
		TurnIndex: 1, UtteranceID: "utt-1", Kind: TranscriptFinal, Text: "你好，我想应聘后端工程师岗位。", Language: "zh-CN", Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("追加 final 失败: %v", err)
	}
	if final.Frozen || final.RevisionState != RevisionNone {
		t.Fatalf("final 初始不应冻结/修订: %+v", final)
	}
	// 修订（下一主问题前允许）。
	revised, err := e.svc.SubmitRevision(context.Background(), testActor, sess.SessionID, RevisionInput{
		RevisionID: "rev-1", UtteranceID: "utt-1", TurnIndex: 1, RevisedText: "你好，我想应聘后端工程师（偏平台方向）岗位。",
	}, "idem-rev-1")
	if err != nil {
		t.Fatalf("提交修订失败: %v", err)
	}
	if revised.RevisionState != RevisionSubmitted {
		t.Fatalf("修订状态应为 submitted: %+v", revised)
	}
	// 冻结回合（turn.completed）。
	res, err := e.svc.FreezeTurn(context.Background(), testActor, sess.SessionID, 1, "idem-freeze-1")
	if err != nil {
		t.Fatalf("冻结回合失败: %v", err)
	}
	if res.FinalCount != 1 || res.RevisedCount != 1 {
		t.Fatalf("冻结统计错误: %+v", res)
	}
	// 冻结后修订状态应升级为 accepted（评分证据）。
	items, err := e.svc.ListTranscripts(context.Background(), testActor, sess.SessionID)
	if err != nil {
		t.Fatalf("列出转写失败: %v", err)
	}
	if len(items) != 1 || items[0].RevisionState != RevisionAccepted || !items[0].Frozen {
		t.Fatalf("冻结后转写应为 accepted + frozen: %+v", items)
	}
}

// TASK-023 异常路径：进入下一主问题（回合冻结）后修订被拒 window_closed。
func TestTranscriptRevisionWindowClosed(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t023-create-2")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if _, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, AppendTranscriptInput{
		TurnIndex: 1, UtteranceID: "utt-2", Kind: TranscriptFinal, Text: "原始回答", Language: "zh-CN", Confidence: 0.85,
	}); err != nil {
		t.Fatalf("追加 final 失败: %v", err)
	}
	if _, err := e.svc.FreezeTurn(context.Background(), testActor, sess.SessionID, 1, "idem-freeze-2"); err != nil {
		t.Fatalf("冻结回合失败: %v", err)
	}
	_, err = e.svc.SubmitRevision(context.Background(), testActor, sess.SessionID, RevisionInput{
		RevisionID: "rev-late", UtteranceID: "utt-2", TurnIndex: 1, RevisedText: "试图改写冻结回答",
	}, "idem-rev-late")
	if !errors.Is(err, ErrRevisionWindowClosed) {
		t.Fatalf("应拒绝窗口外修订，got: %v", err)
	}
	items, _ := e.svc.ListTranscripts(context.Background(), testActor, sess.SessionID)
	if items[0].RevisionState != RevisionRejected || items[0].RevisionRejectedReason != "window_closed" {
		t.Fatalf("修订应记录 rejected(window_closed): %+v", items[0])
	}
	// 冻结后禁止追加 final。
	if _, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, AppendTranscriptInput{
		TurnIndex: 1, UtteranceID: "utt-2", Kind: TranscriptFinal, Text: "覆盖冻结文本", Language: "zh-CN", Confidence: 0.9,
	}); !errors.Is(err, ErrRevisionWindowClosed) {
		t.Fatalf("冻结后追加 final 应被拒绝，got: %v", err)
	}
}

// TASK-023 幂等：同一修订键重放返回首次结果；重复冻结返回 ErrTurnAlreadyFrozen。
func TestTranscriptIdempotency(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t023-create-3")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if _, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, AppendTranscriptInput{
		TurnIndex: 1, UtteranceID: "utt-3", Kind: TranscriptFinal, Text: "回答", Language: "zh-CN", Confidence: 0.8,
	}); err != nil {
		t.Fatalf("追加 final 失败: %v", err)
	}
	first, err := e.svc.SubmitRevision(context.Background(), testActor, sess.SessionID, RevisionInput{
		RevisionID: "rev-idem", UtteranceID: "utt-3", TurnIndex: 1, RevisedText: "修订文本",
	}, "idem-key-1")
	if err != nil {
		t.Fatalf("首次修订失败: %v", err)
	}
	second, err := e.svc.SubmitRevision(context.Background(), testActor, sess.SessionID, RevisionInput{
		RevisionID: "rev-idem", UtteranceID: "utt-3", TurnIndex: 1, RevisedText: "修订文本",
	}, "idem-key-1")
	if err != nil {
		t.Fatalf("重放修订不应报错: %v", err)
	}
	if first.RevisionID != second.RevisionID || first.RevisedText != second.RevisedText {
		t.Fatalf("幂等重放结果应一致: %+v vs %+v", first, second)
	}
	if _, err := e.svc.FreezeTurn(context.Background(), testActor, sess.SessionID, 1, "idem-freeze-3"); err != nil {
		t.Fatalf("冻结回合失败: %v", err)
	}
	if _, err := e.svc.FreezeTurn(context.Background(), testActor, sess.SessionID, 1, "idem-freeze-3-b"); !errors.Is(err, ErrTurnAlreadyFrozen) {
		t.Fatalf("重复冻结应返回 ErrTurnAlreadyFrozen，got: %v", err)
	}
}

// TASK-023 校验：非法 kind / 空字段 / 置信度越界 / 他人会话。
func TestTranscriptValidation(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t023-create-4")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	bad := []AppendTranscriptInput{
		{TurnIndex: 1, UtteranceID: "u", Kind: "bogus", Text: "x", Language: "zh-CN", Confidence: 0.5},
		{TurnIndex: 1, UtteranceID: "", Kind: TranscriptFinal, Text: "x", Language: "zh-CN", Confidence: 0.5},
		{TurnIndex: 1, UtteranceID: "u", Kind: TranscriptFinal, Text: "", Language: "zh-CN", Confidence: 0.5},
		{TurnIndex: 1, UtteranceID: "u", Kind: TranscriptFinal, Text: "x", Language: "", Confidence: 0.5},
		{TurnIndex: 1, UtteranceID: "u", Kind: TranscriptFinal, Text: "x", Language: "zh-CN", Confidence: 1.2},
	}
	for i, in := range bad {
		if _, err := e.svc.AppendTranscript(context.Background(), testActor, sess.SessionID, in); !errors.Is(err, ErrTranscriptInvalid) {
			t.Fatalf("用例 %d 应返回 ErrTranscriptInvalid，got: %v", i, err)
		}
	}
	other := project.Actor{UserID: "user-other", DataRegion: "cn"}
	if _, err := e.svc.ListTranscripts(context.Background(), other, sess.SessionID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("他人会话应返回 ErrNotFound，got: %v", err)
	}
}
