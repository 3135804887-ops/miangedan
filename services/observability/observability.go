// Package observability 提供 OpenTelemetry 观测基线（TASK-005）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-005；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// docs/security/SECURITY-REQUIREMENTS.md SEC-032、SEC-033；docs/observability/LOGGING-POLICY.md。
package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"miangedan/services/region"
)

// 脱敏模式常量。
const (
	// RedactionStrict 为默认脱敏模式；生产环境强制。
	RedactionStrict = "strict"
	// RedactionOff 仅允许非生产环境显式关闭（SDK 级过滤缺失）。
	RedactionOff = "off"
)

// 导出器常量。
const (
	ExporterOTLP = "otlp"
	ExporterNone = "none"
)

// RedactedMarker 为敏感内容被替换后的统一占位符。
const RedactedMarker = "[REDACTED]"

// Config 描述单个服务进程的观测配置（TASK-005）。
type Config struct {
	ServiceName     string
	ServiceVersion  string
	DataRegion      string
	ServiceEnv      string
	LogLevel        string
	OTLPEndpoint    string
	MetricsExporter string
	TracesExporter  string
	RedactionMode   string
}

// Defaults 返回安全默认配置（strict 脱敏、otlp 导出、info 日志）。
func Defaults(serviceName, dataRegion, serviceEnv string) Config {
	return Config{
		ServiceName:     serviceName,
		ServiceVersion:  "unknown",
		DataRegion:      dataRegion,
		ServiceEnv:      serviceEnv,
		LogLevel:        "info",
		MetricsExporter: ExporterOTLP,
		TracesExporter:  ExporterOTLP,
		RedactionMode:   RedactionStrict,
	}
}

// Validate 校验观测配置，任何非法输入 fail-closed（ADR-0005、SEC-032）。
func (c Config) Validate() error {
	if strings.TrimSpace(c.ServiceName) == "" {
		return errors.New("SERVICE_NAME 为空：观测配置缺少服务名")
	}
	if err := region.ValidateDataRegion(c.DataRegion); err != nil {
		return err
	}
	if err := region.ValidateEnvironment(c.ServiceEnv); err != nil {
		return err
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL %q 非法：必须为 debug | info | warn | error", c.LogLevel)
	}
	switch c.RedactionMode {
	case RedactionStrict:
	case RedactionOff:
		if c.ServiceEnv == "production" {
			return errors.New("生产环境禁止关闭日志脱敏（REDACTION_MODE=off 被拒，SEC-032）")
		}
	default:
		return fmt.Errorf("REDACTION_MODE %q 非法：必须为 %s | %s", c.RedactionMode, RedactionStrict, RedactionOff)
	}
	for name, exporter := range map[string]string{
		"metrics": c.MetricsExporter,
		"traces":  c.TracesExporter,
	} {
		if exporter != ExporterOTLP && exporter != ExporterNone {
			return fmt.Errorf("OTEL_%s_EXPORTER %q 非法：必须为 %s | %s", strings.ToUpper(name), exporter, ExporterOTLP, ExporterNone)
		}
	}
	if c.MetricsExporter == ExporterOTLP || c.TracesExporter == ExporterOTLP {
		u, err := url.Parse(c.OTLPEndpoint)
		if err != nil {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT 非法: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT 协议必须为 http/https，实际 %q", u.Scheme)
		}
		if u.Host == "" {
			return errors.New("OTEL_EXPORTER_OTLP_ENDPOINT 缺少主机名")
		}
	}
	return nil
}

// ---------- SDK 级脱敏（SEC-032） ----------

// sensitiveKeyTokens 为键名中的敏感标记；命中即整值替换（fail-closed）。
var sensitiveKeyTokens = []string{
	"token", "secret", "password", "passwd", "credential", "authorization", "key",
	"cookie", "otp", "resume", "transcript", "answer", "raw",
}

// sensitiveValuePatterns 为值中的敏感内容模式：JWT、Bearer、sk- 密钥、超长不透明令牌。
var sensitiveValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/\-=]{16,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{16,}`),
	regexp.MustCompile(`[A-Za-z0-9]{32,}`),
}

// IsSensitiveKey 判断日志/属性键名是否命中敏感标记（SDK 级脱敏规则）。
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, token := range sensitiveKeyTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// RedactString 对字符串应用敏感值模式替换，命中部分统一替换为占位符。
func RedactString(s string) string {
	out := s
	for _, re := range sensitiveValuePatterns {
		out = re.ReplaceAllString(out, RedactedMarker)
	}
	return out
}

// redactAttr 递归脱敏单个日志属性；组内子属性逐项处理。
func redactAttr(a slog.Attr) slog.Attr {
	if a.Equal(slog.Attr{}) {
		return a
	}
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, RedactedMarker)
	}
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, RedactString(a.Value.String()))
	case slog.KindGroup:
		children := a.Value.Group()
		redacted := make([]slog.Attr, 0, len(children))
		for _, child := range children {
			redacted = append(redacted, redactAttr(child))
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(redacted...)}
	default:
		return a
	}
}

// redactHandler 为 SDK 级脱敏日志处理器（strict 模式生效，SEC-032）。
type redactHandler struct {
	inner slog.Handler
}

func (h *redactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, RedactString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, nr)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		redacted = append(redacted, redactAttr(a))
	}
	return &redactHandler{inner: h.inner.WithAttrs(redacted)}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{inner: h.inner.WithGroup(name)}
}

// NewLogger 创建结构化 JSON 日志器；strict 模式自动套用 SDK 级脱敏。
func NewLogger(cfg Config) (*slog.Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	if cfg.RedactionMode == RedactionOff {
		return slog.New(handler), nil
	}
	return slog.New(&redactHandler{inner: handler}), nil
}

// ---------- 指标/追踪属性白名单（PRD Observability and Operations） ----------

// allowedLabels 为指标与追踪属性白名单；白名单外一律拒绝（匿名技术标识）。
var allowedLabels = map[string]bool{
	"data_region":     true,
	"language":        true,
	"input_mode":      true,
	"provider":        true,
	"job_family":      true,
	"version":         true,
	"service_name":    true,
	"service_version": true,
	"service_env":     true,
	"component":       true,
	"operation":       true,
	"status":          true,
	"error_code":      true,
	"queue":           true,
	"workflow_domain": true,
}

// AllowedLabels 返回按字典序排序的白名单（供校验套件与文档核对）。
func AllowedLabels() []string {
	keys := make([]string, 0, len(allowedLabels))
	for k := range allowedLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ValidateAttributes 校验指标/追踪属性集合：白名单外、敏感键或疑似敏感值一律拒绝。
func ValidateAttributes(attrs map[string]string) error {
	if len(attrs) == 0 {
		return errors.New("指标/追踪属性集合为空：必须携带 data_region 等技术标识")
	}
	hasRegion := false
	for key, value := range attrs {
		lower := strings.ToLower(key)
		if lower == "data_region" {
			hasRegion = true
		}
		if !allowedLabels[lower] {
			return fmt.Errorf("属性 %q 不在白名单内（允许：%s）", key, strings.Join(AllowedLabels(), ", "))
		}
		if IsSensitiveKey(lower) {
			return fmt.Errorf("属性 %q 命中敏感键规则，禁止作为标签/追踪属性（SEC-032）", key)
		}
		if RedactString(value) != value {
			return fmt.Errorf("属性 %q 的值疑似包含敏感内容，禁止作为标签/追踪属性（SEC-032）", key)
		}
	}
	if !hasRegion {
		return errors.New("指标/追踪属性缺少 data_region（ADR-0005 容器标签要求）")
	}
	return nil
}

// ---------- OpenTelemetry 装配 ----------

func newResource(cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", cfg.ServiceVersion),
		attribute.String("deployment.environment", cfg.ServiceEnv),
		attribute.String("data_region", cfg.DataRegion),
	}
	return resource.NewSchemaless(attrs...), nil
}

// ShutdownFunc 关闭 OTel 导出器与 Provider；可安全重复调用。
type ShutdownFunc func(context.Context) error

// Setup 创建日志器并按配置装配 OTel 指标/追踪 Provider；
// 任一导出器为 none 时跳过对应 Provider（业务链路不受观测降级影响）。
func Setup(ctx context.Context, cfg Config) (*slog.Logger, ShutdownFunc, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	logger, err := NewLogger(cfg)
	if err != nil {
		return nil, nil, err
	}
	var shutdowns []ShutdownFunc
	if cfg.TracesExporter == ExporterOTLP {
		exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint))
		if err != nil {
			return nil, nil, fmt.Errorf("初始化 OTLP trace 导出器失败: %w", err)
		}
		res, err := newResource(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("初始化 OTel 资源失败: %w", err)
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, func(ctx context.Context) error {
			return errors.Join(tp.Shutdown(ctx), exp.Shutdown(ctx))
		})
	}
	if cfg.MetricsExporter == ExporterOTLP {
		exp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(cfg.OTLPEndpoint))
		if err != nil {
			return nil, nil, fmt.Errorf("初始化 OTLP metric 导出器失败: %w", err)
		}
		res, err := newResource(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("初始化 OTel 资源失败: %w", err)
		}
		mp := metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(exp)),
			metric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, func(ctx context.Context) error {
			return errors.Join(mp.Shutdown(ctx), exp.Shutdown(ctx))
		})
	}
	shutdown := func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	return logger, shutdown, nil
}
