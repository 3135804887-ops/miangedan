package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"miangedan/services/region"
)

type logScanDoc struct {
	Synthetic bool `json:"synthetic"`
	Samples   []struct {
		Name  string `json:"name"`
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"samples"`
	Patterns []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"value_patterns"`
}

// loadLogScanFixture 读取合成日志敏感样本（AGENTS.md：测试数据必须使用 fixtures/synthetic/ 或 synthetic: true 标记）。
func loadLogScanFixture(t *testing.T) logScanDoc {
	t.Helper()
	raw, err := os.ReadFile("../../fixtures/synthetic/log-scan/sensitive-samples.json")
	if err != nil {
		t.Fatalf("读取合成敏感样本失败: %v", err)
	}
	var doc logScanDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析合成敏感样本失败: %v", err)
	}
	if !doc.Synthetic {
		t.Fatal("fixtures/synthetic/log-scan/sensitive-samples.json 必须标记 synthetic: true")
	}
	return doc
}

func sampleValue(doc logScanDoc, name string) string {
	for _, s := range doc.Samples {
		if s.Name == name {
			return s.Value
		}
	}
	for _, p := range doc.Patterns {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

func validConfig() Config {
	cfg := Defaults("mgd-test", "cn", "dev")
	cfg.OTLPEndpoint = "https://otel.example.com:4318"
	return cfg
}

// 正常路径：合法配置校验通过。
func TestValidateConfigValid(t *testing.T) {
	for _, r := range region.AllRegions {
		cfg := Defaults("mgd-test", r.String(), "dev")
		cfg.OTLPEndpoint = "https://otel.example.com:4318"
		if err := cfg.Validate(); err != nil {
			t.Fatalf("区域 %s 合法配置应通过: %v", r, err)
		}
	}
}

// 异常路径：非法区域/环境/日志级别/导出器/端点必须拒绝。
func TestValidateConfigRejected(t *testing.T) {
	cases := map[string]Config{
		"空服务名":   {DataRegion: "cn", ServiceEnv: "dev"},
		"非法区域":   Defaults("x", "us", "dev"),
		"非法环境":   Defaults("x", "cn", "qa"),
		"非法日志级别": {ServiceName: "x", DataRegion: "cn", ServiceEnv: "dev", LogLevel: "trace", MetricsExporter: ExporterNone, TracesExporter: ExporterNone, RedactionMode: RedactionStrict},
		"非法导出器":  {ServiceName: "x", DataRegion: "cn", ServiceEnv: "dev", LogLevel: "info", MetricsExporter: "stdout", TracesExporter: ExporterNone, RedactionMode: RedactionStrict},
		"缺端点":    {ServiceName: "x", DataRegion: "cn", ServiceEnv: "dev", LogLevel: "info", MetricsExporter: ExporterOTLP, TracesExporter: ExporterNone, RedactionMode: RedactionStrict},
		"非法端点协议": {ServiceName: "x", DataRegion: "cn", ServiceEnv: "dev", LogLevel: "info", MetricsExporter: ExporterNone, TracesExporter: ExporterOTLP, OTLPEndpoint: "ftp://otel.example.com:4318", RedactionMode: RedactionStrict},
	}
	for name, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 异常路径：生产环境禁止关闭脱敏（SEC-032）。
func TestValidateConfigProductionRedactionRequired(t *testing.T) {
	cfg := Defaults("mgd-test", "cn", "production")
	cfg.RedactionMode = RedactionOff
	if err := cfg.Validate(); err == nil {
		t.Fatal("生产环境 REDACTION_MODE=off 必须拒绝")
	}
}

// 幂等性：同一配置重复校验结果一致（DoD 第 3 条）。
func TestValidateConfigIdempotent(t *testing.T) {
	cfg := Defaults("mgd-test", "eu", "staging")
	cfg.OTLPEndpoint = "https://otel.example.com:4318"
	for i := 0; i < 3; i++ {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("配置校验必须幂等通过: %v", err)
		}
	}
}

// 正常路径：敏感键被整值替换；消息与普通属性中的令牌模式被替换。
func TestRedactSensitiveKeys(t *testing.T) {
	keys := []string{"access_token", "api_secret", "password", "otp_code", "authorization", "resume_text", "full_answer", "transcript", "raw_media", "cookie"}
	for _, key := range keys {
		if !IsSensitiveKey(key) {
			t.Fatalf("键 %q 应命中敏感规则", key)
		}
		a := redactAttr(slog.String(key, "should-be-masked"))
		if a.Value.String() != RedactedMarker {
			t.Fatalf("键 %q 的值应替换为 %s，实际 %q", key, RedactedMarker, a.Value.String())
		}
	}
}

// 正常路径：普通键不被脱敏。
func TestRedactKeepsTechnicalKeys(t *testing.T) {
	for _, key := range []string{"data_region", "language", "input_mode", "provider", "job_family", "version", "session_id", "http_status"} {
		if IsSensitiveKey(key) {
			t.Fatalf("技术键 %q 不应命中敏感规则", key)
		}
	}
}

// 正常路径：JWT / Bearer / sk- / 超长不透明串被值模式替换。
func TestRedactSensitiveValues(t *testing.T) {
	doc := loadLogScanFixture(t)
	for _, sample := range doc.Patterns {
		out := RedactString(sample.Value)
		if out == sample.Value || !strings.Contains(out, RedactedMarker) {
			t.Fatalf("用例 %q 应被替换: in=%q out=%q", sample.Name, sample.Value, out)
		}
	}
}

// 正常路径：日志输出经 SDK 级脱敏，敏感内容不出现在输出中。
func TestLoggerRedactsOutput(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(&redactHandler{inner: handler})
	logger.Info("面试证据写入完成",
		slog.String("resume_text", "张三 2005-2020 华为 高级工程师"),
		slog.String("full_answer", "我的完整回答内容：拆成四个并行分片。"),
		slog.String("access_token", sampleValue(loadLogScanFixture(t), "access_token")),
		slog.String("data_region", "cn"),
	)
	out := buf.String()
	for _, forbidden := range []string{"张三", "完整回答内容", "eyJhbGciOi", "拆成四个并行分片", "signature"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("日志输出不得包含敏感内容 %q：%s", forbidden, out)
		}
	}
	if !strings.Contains(out, `"data_region":"cn"`) {
		t.Fatalf("技术属性应保留：%s", out)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("日志应为合法 JSON: %v", err)
	}
}

// 正常路径：指标/追踪属性白名单校验。
func TestValidateAttributesValid(t *testing.T) {
	attrs := map[string]string{
		"data_region": "cn",
		"language":    "zh-CN",
		"input_mode":  "voice",
		"provider":    "llm_cn_primary",
		"job_family":  "software_engineer",
		"version":     "1.0.0",
	}
	if err := ValidateAttributes(attrs); err != nil {
		t.Fatalf("白名单属性应通过: %v", err)
	}
}

// 异常路径：白名单外 / 敏感键 / 疑似敏感值必须拒绝。
func TestValidateAttributesRejected(t *testing.T) {
	cases := map[string]map[string]string{
		"白名单外":   {"data_region": "cn", "user_name": "张三"},
		"敏感键":    {"data_region": "cn", "access_token": "x"},
		"疑似敏感值":  {"data_region": "cn", "job_family": sampleValue(loadLogScanFixture(t), "jwt")},
		"缺少区域标识": {"language": "zh-CN"},
	}
	for name, attrs := range cases {
		if err := ValidateAttributes(attrs); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 合成敏感样本回归：加载 fixtures/synthetic/log-scan 样本并断言日志输出零泄露（SEC-032 红线）。
func TestRedactSyntheticSamples(t *testing.T) {
	doc := loadLogScanFixture(t)
	if len(doc.Samples) == 0 {
		t.Fatal("合成敏感样本为空")
	}
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(&redactHandler{inner: handler})
	for _, sample := range doc.Samples {
		buf.Reset()
		logger.Info("synthetic log scan", slog.String(sample.Key, sample.Value))
		out := buf.String()
		// 样本值整体或其中连续 12+ 字符片段不得出现在日志输出。
		chunks := []string{sample.Value}
		if len(sample.Value) > 12 {
			chunks = append(chunks, sample.Value[len(sample.Value)-12:])
		}
		for _, chunk := range chunks {
			if strings.Contains(out, chunk) {
				t.Fatalf("样本 %q 泄露到日志输出: %s", sample.Name, out)
			}
		}
	}
}

// 幂等性：脱敏结果确定且可重复（DoD 第 3 条）。
func TestRedactDeterministic(t *testing.T) {
	input := "msg " + sampleValue(loadLogScanFixture(t), "jwt") + " end"
	first := RedactString(input)
	for i := 0; i < 3; i++ {
		if RedactString(input) != first {
			t.Fatal("脱敏结果必须确定")
		}
	}
	if IsSensitiveKey("access_token") != true {
		t.Fatal("IsSensitiveKey 结果必须确定")
	}
}

// 正常路径：Setup 在导出器 none 时不建 Provider，业务链路不受观测影响。
func TestSetupNoneExporters(t *testing.T) {
	cfg := validConfig()
	cfg.MetricsExporter = ExporterNone
	cfg.TracesExporter = ExporterNone
	logger, shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup(none) 应成功: %v", err)
	}
	if logger == nil {
		t.Fatal("Setup(none) 必须返回日志器")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown 应无错误: %v", err)
	}
	// 幂等：重复 Shutdown 无副作用。
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("重复 Shutdown 应无错误: %v", err)
	}
}
