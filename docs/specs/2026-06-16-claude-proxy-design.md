# claude-proxy 设计文档 V1.0

## 1. 项目目标

`claude-proxy` 是一个透明的本地中间人进程，伪装成 `claude` 可执行入口，拦截并转发 VS Code Claude 扩展与真实 Claude Code CLI 之间的通信。

核心目标：

1. 不破坏 VS Code Claude 扩展与 Claude Code 的原有调用链路
2. 旁路捕获 Claude Code 会话流，提取消息、工具调用、diff、session_id、cost 等信息
3. 通过本地 WebSocket 暴露给 `cc-connect`
4. 支持微信端通过 `cc-connect` 查看会话并发送受控指令
5. 在任何异常情况下，优先保证 Claude Code 主链路继续可用

**核心原则：桥接能力可以失败，Claude Code 主链路不能失败。**

---

## 2. 总体架构

```
VS Code Claude 扩展 ──(pty/pipe)──▶ claude-proxy ──(pty/pipe)──▶ 真实 claude.exe
                                         │
                              ws://127.0.0.1:9876?token=<uuid>
                                         │
                                     cc-connect
                                         │
                                       微信
```

`claude-proxy` 位于 PATH 优先级更高的位置，代理进程内部通过配置文件中的绝对路径启动真实 Claude Code CLI。

---

## 3. 核心设计原则

### 3.1 完全透明

完整透传：argv、env、cwd、stdin、stdout、stderr、exit code、signal/interrupt、终端尺寸变化、Ctrl+C/D 等控制输入。

### 3.2 主链路优先

WebSocket、JSON 解析、日志、事件提取都属于旁路能力。任何旁路模块失败不得影响 VS Code 扩展继续使用 Claude Code。

### 3.3 不主动改变 Claude 调用方式

不主动附加 `--output-format stream-json`、`--input-format stream-json` 等参数，除非未来通过显式配置开启。

---

## 4. 行为清单

| # | 行为 | 说明 |
|---|------|------|
| 1 | 透明转发 | argv/env/cwd/stdin/stdout/stderr/exit code 完全透传 |
| 2 | PTY/pipe 自动识别 | 交互场景 ConPTY/PTY，非交互 pipe |
| 3 | 原始流永远透传 | stdout/stderr 不经修改 |
| 4 | JSON 尽力解析 | 检测到 NDJSON 则解析，连续失败达阈值则停止 |
| 5 | WebSocket 旁路 | 只读取、订阅、注入受控指令 |
| 6 | stdin 注入受控 | 仅 idle/awaiting_user_input 状态接受普通消息 |
| 7 | 故障隔离 | 解析/WS/日志失败均不影响 Claude 主流程 |
| 8 | 防递归 | 真实 Claude 路径通过绝对路径配置 |
| 9 | 鉴权 | 本地随机 token，cc-connect 读取后连接 |
| 10 | 脱敏 | API key/token/cookie 等敏感信息不写入日志 |

---

## 5. 进程启动流程

1. VS Code 扩展调用 `claude` → 因 PATH 优先级进入 `claude-proxy`
2. 代理读取 `~/.cc-connect/claude-proxy/config.json`
3. 防递归检查：real_bin 存在、绝对路径、不等于自身、可执行
4. 代理使用原始 argv/env/cwd/stdin 启动真实 Claude

配置示例：

```json
{
  "real_bin": "C:\\Users\\xxx\\AppData\\Roaming\\npm\\claude.cmd",
  "ws_host": "127.0.0.1",
  "ws_port": 9876,
  "log_level": "info",
  "enable_json_parse": true,
  "enable_ws": true
}
```

---

## 6. PTY / pipe 模式

### pipe mode
适用于非交互命令、CI、stream-json 场景。实现简单，易于解析结构化流。

### pty mode
适用于 VS Code 扩展交互调用。Windows 走 ConPTY，macOS/Linux 走 PTY。优先保证交互行为正确，JSON 解析为附加能力。

---

## 7. 流处理

- **原始流**：stdout/stderr 直接转发，不修改、不过滤、不重排
- **旁路解析**：复制一份流进入解析器 → event bus → WebSocket
- **JSON 尽力解析**：按行识别 NDJSON，独立解析每行。成功提取事件，连续失败达阈值后停止结构化解析

可提取事件：session_id、user/assistant message、partial message、tool_use、tool_result、diff、cost、permission request、status update、error event

---

## 8. WebSocket 设计

- 仅监听 `127.0.0.1:9876`
- 连接地址：`ws://127.0.0.1:9876/ws?token=<uuid>`
- 启动时生成随机 token，写入 `~/.cc-connect/claude-proxy/runtime/token`
- 校验：token、Origin、来源 IP(127.0.0.1)、session_id

**推送事件：**
```json
{"type": "event", "session_id": "xxx", "event": "assistant_message", "payload": {}}
```

**注入消息：**
```json
{"type": "inject", "session_id": "xxx", "content": "继续执行"}
```

**控制指令：**
```json
{"type": "control", "session_id": "xxx", "command": "status|cancel|approve|reject|ping"}
```

---

## 9. stdin 注入

- **普通消息**：仅 idle / awaiting_user_input 状态允许
- **非 idle 状态**（running_tool / awaiting_permission / streaming_response）：仅允许 cancel/approve/reject/status
- **语义约定**：stdin 注入只等价于向 Claude Code 进程写入输入，不等价于 VS Code UI 输入框打字

---

## 10. 故障隔离

| 场景 | 行为 |
|------|------|
| 代理启动失败但 real_bin 可用 | 直接 exec 真实 Claude |
| JSON 解析失败 | 停止结构化解析，继续原始流转发 |
| WebSocket 模块异常 | 关闭 WS 模块，不中断 Claude |
| 日志写入失败 | 禁用日志，不影响主链路 |
| 真实 Claude 启动失败 | 原样返回 stderr 和 exit code |

---

## 11. 安全

- 仅监听 127.0.0.1
- 每次启动生成随机 token，不写入普通日志
- 日志脱敏：API key、token、cookie、authorization header、SSH key 等
- V1 默认不保存完整会话正文

---

## 12. Windows 安装

1. 扫描 PATH 中所有 `claude*` 入口
2. 找到真实 Claude Code 入口写入 `config.json`
3. `claude-proxy` 所在目录置于 PATH 最前
4. 必要时同时提供 `claude.exe`/`claude.cmd` shim
5. 代理内部绝不通过 PATH 再次查找 `claude`

---

## 13. 技术选型

Go 单二进制。cc-connect 同语言可复用。核心模块：

```
cmd/claude-proxy
internal/config
internal/launcher
internal/stream
internal/parser
internal/ws
internal/session
internal/security
internal/logging
internal/platform/pty
```

---

## 14. 开发阶段

### Phase 0：探针验证
确认 VS Code Claude 扩展实际调用方式、argv/env/cwd、pipe/pty、stdout/stderr 是否含 JSON。

### Phase 1：透明代理
完成真实 Claude 启动、全量透传、防递归配置。

### Phase 2：WebSocket 旁路
本地 WS、token 鉴权、cc-connect 连接、session 订阅、事件推送。

### Phase 3：JSON 事件解析
NDJSON 识别、事件提取、解析失败降级。

### Phase 4：受控 stdin 注入
session state gate、idle 注入、控制指令。

### Phase 5：安全和安装
日志脱敏、Windows/macOS/Linux shim、安装脚本。

---

## 15. V1 验收标准

**主链路：** VS Code Claude 正常启动/对话/工具调用/diff/Ctrl+C/退出码，代理关闭旁路仍可用。

**WebSocket：** cc-connect 通过 token 连接，订阅 session，接收事件，断开不影响主链路。

**注入：** idle 可注入，非 idle 拒绝普通消息，cancel/status 可用，注入失败不破坏主链路。

**故障：** JSON 解析失败后仍可用，WS 异常后仍可用，日志失败后仍可用，配置缺失返回明确错误。

---

## 16. 非目标项（V1 不做）

1. VS Code 扩展
2. 读取 VS Code 编辑器内部状态（打开文件、光标位置等）
3. 主动改变 Claude Code 参数
4. 强制 Claude Code 使用 stream-json
5. 覆盖真实 Claude Code 二进制
6. 监听公网地址
7. 保存完整明文会话日志
8. 获取模型隐藏 chain-of-thought

---

## 17. Phase 0 实测结果（2026-06-16）

### 入口方式

VS Code Claude 扩展通过 `claudeCode.claudeProcessWrapper` 设置指定代理路径。代理二进制放在 `~/.cc-connect/claude-proxy/bin/claude.exe`，`real_bin` 指向扩展自带二进制。

### 实测数据

| 项目 | 值 |
|------|-----|
| argv | `--output-format stream-json --verbose --input-format stream-json --include-partial-messages --replay-user-messages` 等 17 个参数 |
| cwd | 项目根目录 |
| stdin | pipe（非 terminal） |
| stdout | pipe，纯 NDJSON 流 |
| stderr | pipe，调试日志 |
| VS Code 注入 | args[0] 为原始 `.exe` 路径，已通过 `filterArgs` 过滤 |

### 决策

- **不需要 ConPTY**：全部 pipe 模式
- **不需要猜测输出格式**：确认为 NDJSON stream-json
- **安全加固**：样本写入前通过 `security.RedactString` 正则脱敏 API key/token
