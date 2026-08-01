# 日志与观测脱敏政策（LOGGING-POLICY）

| 字段 | 内容 |
|---|---|
| 文档编号 | OBS-POLICY-001 |
| 版本 | 0.1.0（2026-08-01，TASK-005） |
| 追踪 | PRD 安全控制（日志禁令）与 Observability and Operations；SEC-032；THREAT-MODEL TM-11；PRIVACY-DATA-MAP |
| 适用范围 | 全部运行时（Go 控制面、Python AI 服务、前端错误上报、工作流 Worker） |

## 1. 目的

保证日志、追踪与指标默认脱敏：**日志与追踪零正文，监控零个人标识**。任何运行时产生的日志、
追踪属性或指标标签都不得包含简历正文、完整回答、令牌、原始媒体，也不得包含可识别的面试内容。

## 2. 禁止内容（红线，SEC-032）

1. 简历正文（含用户上传简历的结构化文本或摘要）。
2. 完整回答（含 ASR 最终文本、修订后文本与逐字稿正文）。
3. 令牌（JWT、Bearer、API Key、OAuth/会话令牌、OTP、签名密钥）。
4. 原始媒体（音视频 URI、媒体字节流、授权媒体的引用标识）。

> 合成回归样本见 `fixtures/synthetic/log-scan/sensitive-samples.json`（`synthetic: true`），
> 用于验证 SDK 级脱敏零泄露；禁止把真实简历、回答、手机号、邮箱、证件号写入任何测试材料。

## 3. SDK 级脱敏规则（strict 模式）

`services/observability`（Go 控制面共享包）提供默认开启的 SDK 级过滤，规则如下：

1. **敏感键整值替换**：日志属性键名（小写后）命中以下标记即整值替换为 `[REDACTED]`：
   `token`、`secret`、`password`、`passwd`、`credential`、`authorization`、`cookie`、`otp`、
   `key`、`resume`、`transcript`、`answer`、`raw`。
2. **敏感值模式替换**：消息与普通属性值中的 JWT（`eyJ…` 三段式）、`Bearer <token>`、
   `sk-` 开头密钥、32+ 连续字母数字的不透明令牌统一替换为 `[REDACTED]`。
3. **生产强制 strict**：`REDACTION_MODE=off` 在 `production` 环境直接拒绝启动；`strict` 为默认。
4. 新增运行时（Python/Node）必须实现等价 SDK 级过滤并接入同一合成样本回归，不得绕过本政策。

## 4. 结构化日志字段

允许的日志字段以匿名技术标识为限：`data_region`、`service_name`、`service_version`、
`service_env`、`component`、`operation`、`status`、`error_code`、匿名会话编号、追踪 ID 等。
正文类内容一律不进日志，也不得通过格式化消息拼接进入。

## 5. 指标与追踪属性白名单

指标与追踪属性只允许以下白名单键（`services/observability` 的 `ValidateAttributes` 强制校验）：

`data_region`、`language`、`input_mode`、`provider`、`job_family`、`version`、
`service_name`、`service_version`、`service_env`、`component`、`operation`、`status`、
`error_code`、`queue`、`workflow_domain`。

白名单外、敏感键或疑似敏感值一律拒绝；指标集合必须携带 `data_region`（ADR-0005）。
PRD 要求的按数据区、语言、输入模式、供应商、岗位族、版本切分全部由白名单标签承载。

## 6. 错误上报

错误追踪只上报匿名技术指标、错误码与**脱敏后**的堆栈；错误消息中如出现令牌或正文，
由 SDK 级过滤先替换再上报（PRIVACY-DATA-MAP：禁止正文上报的 SDK 级过滤）。

## 7. 扫描与审计

- CI 阶段 1 的 `observability` 套件与阶段 5 的 `secrets` 套件执行仓内日志敏感数据扫描；
  `services/observability` 单测使用合成敏感样本断言日志输出零泄露（发布门禁）。
- 月度运营执行线上日志/追踪管道敏感数据扫描（SEC-032、PRIVACY-DATA-MAP）。
- 发现正文进入日志/追踪按隐私事故分级处置（SEV 响应、清除、复盘、修订过滤规则）。

## 8. 降级规则

观测链路故障不得影响业务链路；脱敏管道故障时**宁可丢弃日志，不得泄露正文**
（SYSTEM-ARCHITECTURE 第 10 节）。
