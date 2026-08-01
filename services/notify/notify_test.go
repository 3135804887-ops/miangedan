package notify

import (
	"context"
	"testing"

	"miangedan/services/region"
)

func validConfig(regionCode string) Config {
	return Config{
		DataRegion: regionCode,
		ServiceEnv: "dev",
		EmailFrom:  "mgd-" + regionCode + "-dev@miangedan.example",
		Channels:   []string{ChannelEmail},
	}
}

// 正常路径：三区合法配置通过。
func TestNotifyConfigValid(t *testing.T) {
	for _, r := range region.AllRegions {
		if err := validConfig(r.String()).Validate(); err != nil {
			t.Fatalf("区域 %s 合法配置应通过: %v", r, err)
		}
	}
}

// 异常路径：非法区域/环境/发件人/通道必须拒绝。
func TestNotifyConfigRejected(t *testing.T) {
	cases := map[string]Config{
		"非法区域":    {DataRegion: "us", ServiceEnv: "dev", EmailFrom: "a@b.example", Channels: []string{ChannelEmail}},
		"非法环境":    {DataRegion: "cn", ServiceEnv: "qa", EmailFrom: "a@b.example", Channels: []string{ChannelEmail}},
		"空发件人":    {DataRegion: "cn", ServiceEnv: "dev", EmailFrom: "", Channels: []string{ChannelEmail}},
		"非法发件人":   {DataRegion: "cn", ServiceEnv: "dev", EmailFrom: "not-an-email", Channels: []string{ChannelEmail}},
		"未知通道":    {DataRegion: "cn", ServiceEnv: "dev", EmailFrom: "a@b.example", Channels: []string{"sms"}},
		"缺少email": {DataRegion: "cn", ServiceEnv: "dev", EmailFrom: "a@b.example", Channels: []string{}},
	}
	for name, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 正常路径：消息校验通过（同区域、幂等键、模板变量合法）。
func TestMessageValid(t *testing.T) {
	msg := Message{
		DataRegion:     "cn",
		To:             "synthetic@example.com",
		TemplateID:     "otp_login",
		Variables:      map[string]string{"code_expire_minutes": "10"},
		IdempotencyKey: "msg-0001",
	}
	if err := msg.Validate("cn"); err != nil {
		t.Fatalf("合法消息应通过: %v", err)
	}
}

// 异常路径：跨区发送、缺幂等键、变量携带敏感键必须拒绝。
func TestMessageRejected(t *testing.T) {
	cases := map[string]Message{
		"跨区发送":  {DataRegion: "eu", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "msg-0001"},
		"空目标":   {DataRegion: "cn", To: "", TemplateID: "otp_login", IdempotencyKey: "msg-0001"},
		"空模板":   {DataRegion: "cn", To: "synthetic@example.com", TemplateID: "", IdempotencyKey: "msg-0001"},
		"缺幂等键":  {DataRegion: "cn", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: ""},
		"变量含正文": {DataRegion: "cn", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "msg-0001", Variables: map[string]string{"full_answer": "我的完整回答"}},
		"变量含令牌": {DataRegion: "cn", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "msg-0001", Variables: map[string]string{"access_token": "x"}},
	}
	for name, msg := range cases {
		if err := msg.Validate("cn"); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 正常路径：路由器三区就绪，同区消息可达。
func TestRouterSend(t *testing.T) {
	configs := map[string]Config{
		"cn":   validConfig("cn"),
		"eu":   validConfig("eu"),
		"intl": validConfig("intl"),
	}
	router, err := NewRouter(configs)
	if err != nil {
		t.Fatalf("路由器应创建成功: %v", err)
	}
	if len(router.Regions()) != 3 {
		t.Fatalf("应配置三区，实际 %v", router.Regions())
	}
	for _, rc := range []string{"cn", "eu", "intl"} {
		if err := router.Send(context.Background(), Message{DataRegion: rc, To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "k-" + rc}); err != nil {
			t.Fatalf("区域 %s 发送应通过: %v", rc, err)
		}
	}
}

// 异常路径：某区未配置不影响他区；未配置区 fail-closed 拒绝（单区通道故障不影响他区）。
func TestRouterIsolation(t *testing.T) {
	// eu 通道"故障"（从路由器移除）后，cn 仍可发送。
	configs := map[string]Config{"cn": validConfig("cn")}
	router, err := NewRouter(configs)
	if err != nil {
		t.Fatalf("移除 eu 后路由器应仍可创建: %v", err)
	}
	if err := router.Send(context.Background(), Message{DataRegion: "cn", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "k-cn"}); err != nil {
		t.Fatalf("cn 通道应不受影响: %v", err)
	}
	if err := router.Send(context.Background(), Message{DataRegion: "eu", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "k-eu"}); err == nil {
		t.Fatal("未配置的 eu 区域必须 fail-closed 拒绝")
	}
}

// 幂等性：同一配置重复校验/发送结果一致（DoD 第 3 条）。
func TestNotifyIdempotent(t *testing.T) {
	cfg := validConfig("cn")
	for i := 0; i < 3; i++ {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("配置校验必须幂等通过: %v", err)
		}
	}
	router, err := NewRouter(map[string]Config{"cn": cfg})
	if err != nil {
		t.Fatal(err)
	}
	msg := Message{DataRegion: "cn", To: "synthetic@example.com", TemplateID: "otp_login", IdempotencyKey: "k-1"}
	for i := 0; i < 3; i++ {
		if err := router.Send(context.Background(), msg); err != nil {
			t.Fatalf("发送校验必须幂等通过: %v", err)
		}
	}
}
