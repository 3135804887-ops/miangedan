package room

import (
	"context"
	"errors"
	"testing"
	"time"

	"miangedan/services/project"
)

type projectAPIAdapter struct {
	svc *project.Service
}

func (a projectAPIAdapter) GetProject(ctx context.Context, actor project.Actor, id string) (project.Project, error) {
	return a.svc.GetProject(ctx, actor, id)
}
func (a projectAPIAdapter) GetPlan(ctx context.Context, actor project.Actor, id string) (project.PlanVersion, error) {
	return a.svc.GetPlan(ctx, actor, id)
}
func (a projectAPIAdapter) ClaimDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ClaimDevice(ctx, actor, id, device, key)
}
func (a projectAPIAdapter) TransferDevice(ctx context.Context, actor project.Actor, id, current, next, key string) (project.Project, error) {
	return a.svc.TransferDevice(ctx, actor, id, current, next, key)
}
func (a projectAPIAdapter) ReleaseDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ReleaseDevice(ctx, actor, id, device, key)
}

type testEnv struct {
	svc       *Service
	store     *MemoryStore
	projSvc   *project.Service
	projStore *project.MemoryStore
	now       time.Time
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	flow, err := project.LoadFlowConfig("")
	if err != nil {
		t.Fatalf("加载流程配置失败: %v", err)
	}
	projStore := project.NewMemoryStore()
	projSvc, err := project.NewService(projStore, projStore, flow)
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	tokens, err := NewMediaTokenManager(TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: TokenTTLDefault}, store)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(store, store, tokens, StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	svc.now = func() time.Time { return now }
	return &testEnv{svc: svc, store: store, projSvc: projSvc, projStore: projStore, now: now}
}

var testActor = project.Actor{UserID: "user-001", DataRegion: "cn"}

// readyProject 创建 READY 项目且第 1 轮就绪（量表 + 覆盖方案）。
func (e *testEnv) readyProject(t *testing.T, rounds int) project.Project {
	t.Helper()
	proj, err := e.projSvc.CreateProject(context.Background(), testActor, project.CreateInput{
		InterviewLanguage: "zh-CN",
		DegradedMode:      project.ModeFull,
		ResumeRef:         &project.MaterialRef{ID: "resume-1", Version: 1},
	}, "k-"+time.Now().String())
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	plan := project.PlanVersion{
		ProjectID:         proj.ProjectID,
		PlanVersion:       1,
		DataRegion:        proj.DataRegion,
		InterviewLanguage: proj.InterviewLanguage,
		ResumeRef:         proj.ResumeRef,
		JobRef:            proj.JobRef,
		DegradedMode:      proj.DegradedMode,
		RubricVersion:     "rubrics/v1/default",
		DimensionWeights: map[string]int{
			"professional_competence": 25, "problem_solving": 20, "communication": 15,
			"experience_evidence": 15, "behavioral_collaboration": 15, "learning_adaptability": 10,
		},
		Frozen: true,
	}
	for i := 1; i <= rounds; i++ {
		plan.Rounds = append(plan.Rounds, project.RoundConfig{Sequence: i, RoundType: project.RoundTypes[i-1], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{project.DimensionKeys[0]}})
		plan.RoundWeights = append(plan.RoundWeights, project.RoundWeight{Sequence: i, Weight: 100 / rounds})
	}
	if err := e.projStore.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Rounds {
		if err := e.projSvc.SetRoundReadiness(proj.DataRegion, proj.ProjectID, 1, r.Sequence, true, true); err != nil {
			t.Fatalf("标记就绪失败: %v", err)
		}
	}
	proj.Status = project.StatusReady
	proj.PlanVersion = 1
	if err := e.projStore.UpdateProject(proj); err != nil {
		t.Fatal(err)
	}
	return proj
}

// 正常路径：创建会话签发一次性令牌；项目 READY + 设备绑定生效。
func TestCreateSessionOK(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	result, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "create-1")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if result.RoomToken == "" || result.SessionID == "" || result.RoomURL == "" {
		t.Fatalf("会话响应不完整: %+v", result)
	}
	if _, err := e.svc.tokens.Verify(result.RoomToken, e.now.Add(time.Minute)); err != nil {
		t.Fatalf("令牌应可校验: %v", err)
	}
}

// 异常路径：项目非 READY、轮次未就绪、第二台设备、formal_retry 缺 attempt 必须拒绝。
func TestCreateSessionRejected(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	// 非 READY。
	proj.Status = project.StatusDraft
	if err := e.projStore.UpdateProject(proj); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "d"}, "k1"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("非 READY 必须拒绝，实际 %v", err)
	}
	proj.Status = project.StatusReady
	if err := e.projStore.UpdateProject(proj); err != nil {
		t.Fatal(err)
	}
	// 轮次 3 不在计划中。
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 3, Kind: KindFormal, DeviceID: "d"}, "k2"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("计划外轮次必须拒绝，实际 %v", err)
	}
	// 第二台设备（device-a 已被认领）。
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a"}, "k3"); err != nil {
		t.Fatalf("首次认领应成功: %v", err)
	}
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-b"}, "k4"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("第二台设备必须拒绝，实际 %v", err)
	}
	// formal_retry 缺 attempt_id。
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormalRetry, DeviceID: "device-a"}, "k5"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("formal_retry 缺 attempt_id 必须拒绝，实际 %v", err)
	}
}

// 正常/异常路径：结束会话吊销令牌并释放设备；令牌随后被拒。
func TestEndSessionRevokesToken(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	result, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a"}, "c1")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := e.svc.EndSession(context.Background(), testActor, result.SessionID, true, "e1")
	if err != nil || sess.RoomStatus != StatusEnded {
		t.Fatalf("结束会话异常: %v %+v", err, sess)
	}
	if _, err := e.svc.tokens.Verify(result.RoomToken, e.now.Add(time.Minute)); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("结束后令牌必须被吊销，实际 %v", err)
	}
	if _, err := e.svc.EndSession(context.Background(), testActor, result.SessionID, false, "e2"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("未确认退出必须拒绝，实际 %v", err)
	}
}

// 正常/异常路径：3 分钟窗口内重连成功；超窗返回 reconnect_expired。
func TestReconnectWindow(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	result, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a"}, "c1")
	if err != nil {
		t.Fatal(err)
	}
	// 窗口内。
	e.svc.now = func() time.Time { return e.now.Add(time.Minute) }
	reconnected, err := e.svc.ReconnectSession(context.Background(), testActor, result.SessionID, "device-a", 5, "r1")
	if err != nil || reconnected.RoomToken == "" {
		t.Fatalf("窗口内重连应成功: %v", err)
	}
	// 超窗。
	e.svc.now = func() time.Time { return e.now.Add(ReconnectWindow + 2*time.Minute) }
	if _, err := e.svc.ReconnectSession(context.Background(), testActor, result.SessionID, "device-a", 5, "r2"); !errors.Is(err, ErrReconnectExpired) {
		t.Fatalf("超窗必须返回 reconnect_expired，实际 %v", err)
	}
}

// 正常路径：安全转移后新设备令牌可用，旧令牌被吊销。
func TestDeviceTransferSession(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	result, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a"}, "c1")
	if err != nil {
		t.Fatal(err)
	}
	transferred, err := e.svc.DeviceTransferSession(context.Background(), testActor, result.SessionID, "device-c", true, "t1")
	if err != nil {
		t.Fatalf("转移失败: %v", err)
	}
	claims, err := e.svc.tokens.Verify(transferred.RoomToken, e.now.Add(time.Minute))
	if err != nil || claims.DeviceID != "device-c" {
		t.Fatalf("新设备令牌应可用: %v %+v", err, claims)
	}
	if _, err := e.svc.tokens.Verify(result.RoomToken, e.now.Add(time.Minute)); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("旧令牌必须吊销，实际 %v", err)
	}
}

// 幂等性：同键创建返回同一会话（NFR-006）。
func TestCreateSessionIdempotent(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 2)
	in := CreateSessionInput{ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a"}
	first, err := e.svc.CreateSession(context.Background(), testActor, in, "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.svc.CreateSession(context.Background(), testActor, in, "idem-1")
	if err != nil || first.SessionID != second.SessionID {
		t.Fatalf("幂等键应返回同一会话: %v", err)
	}
}
