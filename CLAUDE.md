# CLAUDE.md — claude-proxy

## 技术栈

- Go 1.22+
- 无外部框架依赖，仅标准库 + `golang.org/x/term`（PTY）
- WebSocket 用 `gorilla/websocket` 或 `nhooyr.io/websocket`

## 编码规范

- 包名即目录名，一个包一个职责
- 错误处理：`fmt.Errorf("module: context: %w", err)` 链式包裹
- 日志用 `log/slog`，结构化输出
- 测试文件与源文件同目录，`_test.go` 后缀
- 不做不必要的抽象——先写具体实现，三次复用再抽接口

## 禁止事项

- 禁止代理内部通过 PATH 查找 `claude`（防递归）
- 禁止修改主链路 stdout/stderr 内容
- 禁止监听 0.0.0.0
- 禁止保存完整会话明文日志
- 禁止在 V1 阶段引入 VS Code 扩展 API 依赖
- 禁止主动加 `--output-format stream-json` 等参数

### Phase 1 性能禁令

- 禁止 parser/writer 阻塞 stdout/stderr 主链路
- 禁止使用 `io.MultiWriter` 把 parser writer 接入主链路
- 禁止 stderr 全量写入 events
- 禁止非 JSON 行写成 unknown 事件
- 禁止默认落盘 unknown 事件（`drop_unknown_events` 默认 true）
- 禁止完整写入 raw/tool_result/diff（默认限长 8KB）
- 禁止 Claude Code 读取 `events/*.jsonl` 后由 proxy 再次捕获（日志放大）
- parser、event writer、redactor 失败时必须静默降级

## 核心原则（设计约束）

> 桥接能力可以失败，Claude Code 主链路不能失败。

- JSON 解析失败 → 停解析，继续转发
- WebSocket 死 → 不中断 Claude
- 日志失败 → 禁用日志，不中断
- 启动失败 → 直接 exec 真实 Claude

## 修改规则

- 改 internal 包前先确认不影响 cmd 入口的透明转发语义
- 新增配置项必须设默认值，不强制用户填写
- 新增 WebSocket 消息类型必须在 `internal/ws/messages.go` 注册

## 输出位置

- proxy 二进制：`~/.cc-connect/claude-proxy/bin/claude.exe`
- bridge 二进制：`~/.cc-connect/claude-proxy/bin/bridge.exe`
- 配置：`~/.cc-connect/claude-proxy/config.json`
- 事件日志：`~/.cc-connect/claude-proxy/events/session-*.jsonl`
- 探针日志：`~/.cc-connect/claude-proxy/logs/probe-*.json`

## bridge 运行

bridge 是独立进程，轮询 events JSONL 文件，通过 `cc-connect send` 转发到飞书。
需要环境变量 `BRIDGE_SESSION`（飞书 session key）。
不依赖 WebSocket relay（已知 relay 有事件推送 bug）。

## 深入文档

| 文档 | 内容 |
|------|------|
| [设计文档](docs/specs/2026-06-16-claude-proxy-design.md) | 完整架构、行为清单、验收标准 |
| [CONTEXT.md](CONTEXT.md) | 项目目标、受众、当前状态 |

---

项目背景详见 CONTEXT.md
