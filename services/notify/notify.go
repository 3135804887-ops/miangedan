// Package notify 提供区域化通知通道契约（TASK-007）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-007；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// PRD FR-027（邮箱验证码先行）；PRIVACY-DATA-MAP（通知模板变量无正文）；ADR-0005。
package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miangedan/services/region"
)

// ChannelEmail 为当前唯一通知通道（邮件）；其余通道随业务任务扩展。
const ChannelEmail = "email"

// AllowedChannels 为已批准通知通道集合。
var AllowedChannels = []string{ChannelEmail}

// Config 描述单个数据区的通知通道配置。
type Config struct {
	DataRegion string
	ServiceEnv string
	EmailFrom  string
	Channels   []string
}

// Validate 校验区域通知配置：区域/环境合法、至少含 email 通道、发件人非空且形如邮箱（fail-closed）。
func (c Config) Validate() error {
	if err := region.ValidateDataRegion(c.DataRegion); err != nil {
		return err
	}
	if err := region.ValidateEnvironment(c.ServiceEnv); err != nil {
		return err
	}
	if strings.TrimSpace(c.EmailFrom) == "" {
		return errors.New("EMAIL_FROM 为空：通知通道必须配置发件人")
	}
	if !strings.Contains(c.EmailFrom, "@") {
		return fmt.Errorf("EMAIL_FROM %q 非法：必须为邮箱地址", c.EmailFrom)
	}
	if len(c.Channels) == 0 {
		return errors.New("通知通道列表为空：必须至少含 email")
	}
	seen := make(map[string]bool, len(c.Channels))
	hasEmail := false
	for _, ch := range c.Channels {
		if !contains(AllowedChannels, ch) {
			return fmt.Errorf("未知通知通道 %q（允许：%s）", ch, strings.Join(AllowedChannels, ", "))
		}
		if seen[ch] {
			return fmt.Errorf("通知通道重复 %q", ch)
		}
		seen[ch] = true
		if ch == ChannelEmail {
			hasEmail = true
		}
	}
	if !hasEmail {
		return errors.New("通知通道必须包含 email（邮箱验证码先行，FR-027）")
	}
	return nil
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

// Message 为通知消息；DataRegion 必须与通道所在区域一致（跨区默认拒绝，ADR-0005）。
type Message struct {
	DataRegion     string
	To             string
	TemplateID     string
	Variables      map[string]string
	IdempotencyKey string
}

// Validate 校验消息：目标/模板/幂等键非空，变量不含正文类敏感键，区域与通道一致。
func (m Message) Validate(channelRegion string) error {
	if err := region.ValidateDataRegion(m.DataRegion); err != nil {
		return err
	}
	if m.DataRegion != channelRegion {
		return fmt.Errorf("消息区域 %q 与通道区域 %q 不一致：跨区发送被拒（ADR-0005）", m.DataRegion, channelRegion)
	}
	if strings.TrimSpace(m.To) == "" {
		return errors.New("通知目标为空")
	}
	if strings.TrimSpace(m.TemplateID) == "" {
		return errors.New("通知模板为空")
	}
	if strings.TrimSpace(m.IdempotencyKey) == "" {
		return errors.New("通知缺少幂等键（写操作必须幂等）")
	}
	for key := range m.Variables {
		lower := strings.ToLower(key)
		for _, token := range []string{"resume", "transcript", "answer", "token", "secret", "password", "media", "raw"} {
			if strings.Contains(lower, token) {
				return fmt.Errorf("通知变量 %q 命中敏感键规则：模板变量不得携带正文/令牌/媒体（PRIVACY-DATA-MAP）", key)
			}
		}
	}
	return nil
}

// Router 按数据区路由通知；某区通道故障/未配置不影响其他区（TASK-007 验收）。
type Router struct {
	byRegion map[string]Config
}

// NewRouter 创建区域通知路由器；任一区域配置非法即整体拒绝（fail-closed）。
func NewRouter(configs map[string]Config) (*Router, error) {
	if len(configs) == 0 {
		return nil, errors.New("通知路由器缺少任何区域配置")
	}
	byRegion := make(map[string]Config, len(configs))
	for regionCode, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("区域 %s 通知配置非法: %w", regionCode, err)
		}
		byRegion[regionCode] = cfg
	}
	return &Router{byRegion: byRegion}, nil
}

// Regions 返回已配置区域列表。
func (r *Router) Regions() []string {
	out := make([]string, 0, len(r.byRegion))
	for rc := range r.byRegion {
		out = append(out, rc)
	}
	return out
}

// Send 校验并路由消息；适配器接入点（真实供应商调用随 EPIC-02 业务任务落地）。
func (r *Router) Send(ctx context.Context, msg Message) error {
	if ctx == nil {
		return errors.New("context 不能为 nil")
	}
	cfg, ok := r.byRegion[msg.DataRegion]
	if !ok {
		return fmt.Errorf("区域 %q 未配置通知通道：发送被拒（fail-closed）", msg.DataRegion)
	}
	return msg.Validate(cfg.DataRegion)
}
