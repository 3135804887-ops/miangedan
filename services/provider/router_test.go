package provider

import (
	"errors"
	"testing"

	"miangedan/services/region"
)

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry(nil)
	infos := []Info{
		{ProviderID: "llm_cn_primary", Capability: CapLLM, DataRegion: "cn", Languages: []string{"zh-CN"}, Role: RolePrimary, Version: "1.2.0"},
		{ProviderID: "llm_cn_secondary", Capability: CapLLM, DataRegion: "cn", Languages: []string{"zh-CN", "en-US"}, Role: RoleSecondary, Version: "1.1.0"},
		{ProviderID: "asr_cn_primary", Capability: CapASR, DataRegion: "cn", Languages: []string{"zh-CN"}, Role: RolePrimary, Version: "2.0.0"},
		{ProviderID: "llm_eu_primary", Capability: CapLLM, DataRegion: "eu", Languages: []string{"en-US"}, Role: RolePrimary, Version: "1.2.0"},
	}
	for _, info := range infos {
		if err := r.Register(info, nil); err != nil {
			t.Fatalf("注册 %s 失败: %v", info.ProviderID, err)
		}
	}
	return r
}

// 正常路径：按区域与能力路由；无语言时选主；语言过滤生效。
func TestRoute(t *testing.T) {
	r := testRegistry(t)
	rt := NewRouter(r)
	got, err := rt.Route(CapLLM, "cn", RouteOptions{})
	if err != nil || got.ProviderID != "llm_cn_primary" {
		t.Fatalf("应路由到 llm_cn_primary: %v %+v", err, got)
	}
	got, err = rt.Route(CapLLM, "cn", RouteOptions{Language: "en-US"})
	if err != nil || got.ProviderID != "llm_cn_secondary" {
		t.Fatalf("en-US 应路由到 secondary: %v %+v", err, got)
	}
}

// 异常路径：主供应商熔断后切 secondary；全部不可用拒绝新会话；跨区不可路由。
func TestRouteFailoverAndIsolation(t *testing.T) {
	r := testRegistry(t)
	rt := NewRouter(r)
	// 主熔断。
	for i := 0; i < 5; i++ {
		rt.RecordFailure("llm_cn_primary")
	}
	got, err := rt.Route(CapLLM, "cn", RouteOptions{})
	if err != nil || got.ProviderID != "llm_cn_secondary" {
		t.Fatalf("主 open 应切 secondary: %v %+v", err, got)
	}
	// 主备都熔断 → 拒绝新会话。
	for i := 0; i < 5; i++ {
		rt.RecordFailure("llm_cn_secondary")
	}
	if _, err := rt.Route(CapLLM, "cn", RouteOptions{}); !errors.Is(err, ErrNoVerifiedProvider) {
		t.Fatalf("主备不可用必须拒绝新会话，实际 %v", err)
	}
	// 跨区：intl 无 llm 供应商。
	if _, err := rt.Route(CapLLM, "intl", RouteOptions{}); !errors.Is(err, ErrNoVerifiedProvider) {
		t.Fatalf("跨区不得静默回退，实际 %v", err)
	}
}

// 正常/异常路径：会话钉扎版本；被停用后 Resolve 返回 ErrPinnedUnavailable（不静默切换）。
func TestPinAndResolve(t *testing.T) {
	r := testRegistry(t)
	rt := NewRouter(r)
	info, err := rt.Route(CapLLM, "cn", RouteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rt.Pin("session-1", info)
	pin, err := rt.Resolve("session-1")
	if err != nil || pin.Version != "1.2.0" {
		t.Fatalf("钉扎解析异常: %v %+v", err, pin)
	}
	if err := r.SetStatus("llm_cn_primary", RoleDisabled); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Resolve("session-1"); !errors.Is(err, ErrPinnedUnavailable) {
		t.Fatalf("被停用后必须 ErrPinnedUnavailable（不静默切换），实际 %v", err)
	}
}

// 正常路径：健康探测；异常路径：未注册/非法注册拒绝。
func TestRegistryHealthAndValidation(t *testing.T) {
	r := NewRegistry(nil)
	info := Info{ProviderID: "tts_cn_primary", Capability: CapTTS, DataRegion: "cn", Languages: []string{"zh-CN"}, Role: RolePrimary, Version: "3.0.0"}
	if err := r.Register(info, func() error { return errors.New("synthetic probe down") }); err != nil {
		t.Fatal(err)
	}
	if err := r.Health("tts_cn_primary"); err == nil {
		t.Fatal("健康探测失败应返回错误")
	}
	if err := r.Register(Info{ProviderID: "unknown_xx_primary", Capability: CapLLM, DataRegion: "cn", Languages: []string{"zh-CN"}, Role: RolePrimary, Version: "1"}, nil); err == nil {
		t.Fatal("非法供应商 ID 必须拒绝")
	}
	if err := r.Register(info, nil); err == nil {
		t.Fatal("重复注册必须拒绝")
	}
}

// 区域枚举校验：注册表按区隔离（ADR-0005）。
func TestProviderRegions(t *testing.T) {
	for _, rc := range region.AllRegions {
		info := Info{ProviderID: "search_" + rc.String() + "_primary", Capability: CapSearch, DataRegion: rc.String(), Languages: []string{"zh-CN", "en-US"}, Role: RolePrimary, Version: "1.0.0"}
		if err := ValidateInfo(info); err != nil {
			t.Fatalf("区域 %s 应可注册: %v", rc, err)
		}
	}
}
