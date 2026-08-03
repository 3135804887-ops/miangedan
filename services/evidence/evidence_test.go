package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func payload(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TASK-026 正常路径：问题实际播放、回答、修订、工具事件全量入库；哈希一致。
func TestAppendAllKinds(t *testing.T) {
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []AppendInput{
		{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindQuestionPlayed,
			EventID: "ev-q", PayloadJSON: payload(t, map[string]any{"question_id": "q1", "played_text": "请介绍你自己"})},
		{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindAnswer,
			EventID: "ev-a", PayloadJSON: payload(t, map[string]any{"answer_id": "a1", "modality": "voice"})},
		{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindRevision,
			EventID: "ev-r", PayloadJSON: payload(t, map[string]any{"revision_id": "r1"})},
		{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindToolEvent,
			EventID: "ev-t", PayloadJSON: payload(t, map[string]any{"tool_event_id": "t1"})},
	}
	for _, in := range inputs {
		entry, err := svc.Append(context.Background(), "cn", in)
		if err != nil {
			t.Fatalf("追加 %s 失败: %v", in.Kind, err)
		}
		if entry.ContentHash != HashPayload(in.PayloadJSON) {
			t.Fatalf("哈希不一致")
		}
	}
	items, err := svc.ListBySession(context.Background(), "cn", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("应入库 4 条，实际 %d", len(items))
	}
}

// TASK-026 幂等：同一 event_id 重放返回首条，无副作用。
func TestAppendIdempotent(t *testing.T) {
	store := NewMemoryStore()
	svc, _ := NewService(store)
	in := AppendInput{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindAnswer,
		EventID: "ev-same", PayloadJSON: payload(t, map[string]any{"answer": "x"})}
	first, err := svc.Append(context.Background(), "cn", in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Append(context.Background(), "cn", in)
	if err != nil {
		t.Fatal(err)
	}
	if first.EvidenceID != second.EvidenceID {
		t.Fatalf("幂等重放应返回同一条目")
	}
	items, _ := svc.ListBySession(context.Background(), "cn", "s1")
	if len(items) != 1 {
		t.Fatalf("重放后应仍为 1 条，实际 %d", len(items))
	}
}

// TASK-026 校验：非法类型/空 event_id/非法 JSON/区域错误/哈希不匹配 fail-closed。
func TestAppendValidation(t *testing.T) {
	store := NewMemoryStore()
	svc, _ := NewService(store)
	ok := AppendInput{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindAnswer,
		EventID: "ev-v", PayloadJSON: payload(t, map[string]any{"a": 1})}
	if _, err := svc.Append(context.Background(), "eu", ok); err != nil {
		t.Fatalf("合法输入应成功: %v", err)
	}
	if _, err := svc.Append(context.Background(), "us", ok); err == nil {
		t.Fatal("非法区域应被拒")
	}
	badKind := ok
	badKind.Kind = "delete_everything"
	if _, err := svc.Append(context.Background(), "cn", badKind); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("非法类型应被拒，got: %v", err)
	}
	badJSON := ok
	badJSON.EventID = "ev-bad-json"
	badJSON.PayloadJSON = json.RawMessage("{not json")
	if _, err := svc.Append(context.Background(), "cn", badJSON); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("非法 JSON 应被拒，got: %v", err)
	}
	if _, err := svc.AppendVerified(context.Background(), "cn", ok, "deadbeef"); !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("哈希不匹配应被拒，got: %v", err)
	}
}

// TASK-026 红线：类型层面不存在更新/删除路径（编译期保证），列表返回只读副本。
func TestAppendOnlyNoMutationPath(t *testing.T) {
	store := NewMemoryStore()
	svc, _ := NewService(store)
	in := AppendInput{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1, Kind: KindToolEvent,
		EventID: "ev-tool", PayloadJSON: payload(t, map[string]any{"t": 1})}
	if _, err := svc.Append(context.Background(), "cn", in); err != nil {
		t.Fatal(err)
	}
	items1, _ := svc.ListBySession(context.Background(), "cn", "s1")
	items2, _ := svc.ListBySession(context.Background(), "cn", "s1")
	items1[0].PayloadJSON = json.RawMessage(`{"tampered": true}`)
	if string(items2[0].PayloadJSON) != `{"t":1}` {
		t.Fatalf("列表应返回副本，篡改不落库: %s", items2[0].PayloadJSON)
	}
}

// failingStore 模拟持久化存储故障（TC-NFR-005-A01）。
type failingStore struct {
	*MemoryStore
}

func (f *failingStore) SaveEntry(Entry) error {
	return errors.New("storage unavailable")
}

// TASK-090 补测（TC-NFR-005-A01）：持久化失败时 Append 报错，不产生部分写入，调用方必须阻塞推进。
func TestAppendStoreFailureBlocksAdvance(t *testing.T) {
	store := &failingStore{MemoryStore: NewMemoryStore()}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	in := AppendInput{SessionID: "s1", TurnIndex: 1, ProjectID: "p1", RoundSeq: 1,
		Kind: KindAnswer, EventID: "ev-answer-1",
		PayloadJSON: payload(t, map[string]any{"answer_id": "a1"})}
	if _, err := svc.Append(context.Background(), "cn", in); err == nil {
		t.Fatal("存储故障时 Append 必须报错（阻塞推进）")
	}
	items, err := svc.ListBySession(context.Background(), "cn", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("失败写入不得产生部分证据，实际 %d 条", len(items))
	}
}
