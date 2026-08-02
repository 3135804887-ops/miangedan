package room

import (
	"context"
	"errors"
	"testing"

	"miangedan/services/project"
)

// readyProjectWithTools 构造第 1 轮已配置 code_editor 的 READY 项目。
func (e *testEnv) readyProjectWithTools(t *testing.T) project.Project {
	t.Helper()
	proj := e.readyProject(t, 1)
	plan, err := e.projSvc.GetPlan(context.Background(), testActor, proj.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	plan.Rounds[0].Tools = []string{"code_editor"}
	if err := e.projStore.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	return proj
}

// TASK-024 正常路径：仅计划已配置工具可激活；事件幂等入库。
func TestToolActivateAndRecord(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProjectWithTools(t)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t024-create")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	act, err := e.svc.ActivateTool(context.Background(), testActor, sess.SessionID, ActivateToolInput{ToolKey: ToolCodeEditor})
	if err != nil {
		t.Fatalf("激活已配置工具失败: %v", err)
	}
	if act.ToolKey != ToolCodeEditor || act.SessionID != sess.SessionID {
		t.Fatalf("激活结果错误: %+v", act)
	}
	ev, err := e.svc.RecordToolEvent(context.Background(), testActor, sess.SessionID, ToolEvent{
		ToolKey: ToolCodeEditor, ToolEventID: "tool-ev-1", EventType: ToolEventRun, ContentRef: "s3://region/uploads/session/tool/1",
	}, "idem-tool-1")
	if err != nil {
		t.Fatalf("记录工具事件失败: %v", err)
	}
	if ev.EventType != ToolEventRun || ev.ContentRef == "" {
		t.Fatalf("工具事件错误: %+v", ev)
	}
	// 幂等重放。
	again, err := e.svc.RecordToolEvent(context.Background(), testActor, sess.SessionID, ToolEvent{
		ToolKey: ToolCodeEditor, ToolEventID: "tool-ev-1", EventType: ToolEventRun, ContentRef: "s3://region/uploads/session/tool/1",
	}, "idem-tool-1")
	if err != nil || again.ToolEventID != ev.ToolEventID {
		t.Fatalf("幂等重放失败: %v %+v", err, again)
	}
	items, err := e.svc.ListToolEvents(context.Background(), testActor, sess.SessionID)
	if err != nil {
		t.Fatalf("列出工具事件失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("幂等后应仅一条事件，实际 %d", len(items))
	}
}

// TASK-024 异常路径：未配置工具激活/事件均被拒；非法类型被拒。
func TestToolNotConfiguredRejected(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1) // 未配置任何工具
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t024-create-2")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if _, err := e.svc.ActivateTool(context.Background(), testActor, sess.SessionID, ActivateToolInput{ToolKey: ToolCodeEditor}); !errors.Is(err, ErrToolNotConfigured) {
		t.Fatalf("未配置工具应拒绝激活，got: %v", err)
	}
	if _, err := e.svc.RecordToolEvent(context.Background(), testActor, sess.SessionID, ToolEvent{
		ToolKey: ToolCodeEditor, ToolEventID: "tool-ev-2", EventType: ToolEventEdit, ContentRef: "s3://ref",
	}, "idem-tool-2"); !errors.Is(err, ErrToolNotConfigured) {
		t.Fatalf("未配置工具事件应被拒，got: %v", err)
	}
	// 非法类型。
	proj2 := e.readyProjectWithTools(t)
	sess2, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj2.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-b",
	}, "t024-create-3")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if _, err := e.svc.ActivateTool(context.Background(), testActor, sess2.SessionID, ActivateToolInput{ToolKey: "hack_tool"}); !errors.Is(err, ErrToolInvalid) {
		t.Fatalf("未知工具应返回 ErrToolInvalid，got: %v", err)
	}
	if _, err := e.svc.RecordToolEvent(context.Background(), testActor, sess2.SessionID, ToolEvent{
		ToolKey: ToolCodeEditor, ToolEventID: "tool-ev-3", EventType: "exfiltrate", ContentRef: "s3://ref",
	}, "idem-tool-3"); !errors.Is(err, ErrToolInvalid) {
		t.Fatalf("未知事件类型应返回 ErrToolInvalid，got: %v", err)
	}
}
