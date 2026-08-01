package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"miangedan/services/source"
)

func TestMemorySaveIdempotent(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	exp := now.Add(30 * 24 * time.Hour)
	url := "https://careers.example.com/hiring-process"
	base := source.ProcessSource{
		SourceID:       "src-test-0001",
		URL:            &url,
		SourceType:     source.OfficialCareersPage,
		RetrievedAt:    now,
		Credibility:    source.CredibilityHigh,
		ExpiresAt:      &exp,
		Region:         "cn",
		JobFamily:      "data_engineering",
		Status:         source.StatusActive,
		IdempotencyKey: "search-key-0001",
		DataRegion:     "cn",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	saved, err := m.Save(ctx, base)
	if err != nil || !saved {
		t.Fatalf("首次保存应成功: saved=%v err=%v", saved, err)
	}
	// 同幂等键重复写入：不产生重复副作用（NFR-006）。
	saved, err = m.Save(ctx, base)
	if err != nil {
		t.Fatalf("重复保存不应报错: %v", err)
	}
	if saved {
		t.Fatal("同幂等键重复保存必须返回 saved=false")
	}
	got, err := m.Get(ctx, "src-test-0001")
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if got.SourceID != base.SourceID {
		t.Fatalf("来源不匹配: %v", got.SourceID)
	}
}

func TestMemorySaveConcurrentIdempotent(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	url := "https://careers.example.com/concurrent"
	base := source.ProcessSource{
		SourceID:       "src-test-0002",
		URL:            &url,
		SourceType:     source.OfficialRecruitingContent,
		RetrievedAt:    now,
		Credibility:    source.CredibilityMedium,
		Region:         "cn",
		JobFamily:      "data_engineering",
		Status:         source.StatusActive,
		IdempotencyKey: "search-key-0002",
		DataRegion:     "cn",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, _ = m.Save(ctx, base)
		}()
	}
	wg.Wait()
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.byID) != 1 || len(m.byKey) != 1 {
		t.Fatalf("并发同键保存必须只落一条：byID=%d byKey=%d", len(m.byID), len(m.byKey))
	}
}

func TestMemoryListFilters(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	exp := now.Add(30 * 24 * time.Hour)
	mk := func(id, region, family string, status source.SourceStatus) source.ProcessSource {
		u := "https://careers.example.com/" + id
		return source.ProcessSource{
			SourceID:       id,
			URL:            &u,
			SourceType:     source.CrediblePublicMaterial,
			RetrievedAt:    now,
			Credibility:    source.CredibilityMedium,
			ExpiresAt:      &exp,
			Region:         region,
			JobFamily:      family,
			Status:         status,
			IdempotencyKey: "key-" + id,
			DataRegion:     region,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	if _, err := m.Save(ctx, mk("a", "cn", "data_engineering", source.StatusActive)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Save(ctx, mk("b", "cn", "software_engineering", source.StatusActive)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Save(ctx, mk("c", "intl", "data_engineering", source.StatusTakenDown)); err != nil {
		t.Fatal(err)
	}
	// 跨区列表拒绝（ADR-0005）。
	if _, err := m.List(ctx, ListQuery{DataRegion: "us"}); err == nil {
		t.Fatal("非法区域列表必须拒绝")
	}
	// 正常路径：区域+岗位族过滤。
	got, err := m.List(ctx, ListQuery{DataRegion: "cn", JobFamily: "data_engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SourceID != "a" {
		t.Fatalf("区域+岗位族过滤结果错误: %+v", got)
	}
	// 异常路径：状态过滤与跨区隔离（intl 数据对 cn 不可见）。
	got, err = m.List(ctx, ListQuery{DataRegion: "cn", Status: source.StatusTakenDown})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("cn 区不应看到 taken_down（该记录在 intl）: %+v", got)
	}
	// 分页：limit 生效。
	got, err = m.List(ctx, ListQuery{DataRegion: "cn", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("limit=1 应只返回一条: %d", len(got))
	}
}

func TestMemoryGetNotFound(t *testing.T) {
	m := NewMemory()
	if _, err := m.Get(context.Background(), "missing"); err != ErrNotFound {
		t.Fatalf("应返回 ErrNotFound: %v", err)
	}
}
