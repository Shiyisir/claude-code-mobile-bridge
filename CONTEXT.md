# CONTEXT.md — claude-proxy

## 项目目标

让 VS Code 里正在运行的 Claude Code 会话能被手机端（飞书/微信）查看。通过 cc-connect 桥接到聊天平台。

## 目标受众

- **用户**：拾易
- **下游系统**：cc-connect（飞书），bridge 进程（JSONL → cc-connect send）

## 当前状态

**Phase 2.5 完成 ✅** — VS Code 会话实时同步到飞书。

### 架构（最终落地版）

```
VS Code Claude → claude-proxy (透明代理) → events/*.jsonl → bridge (2s轮询) → cc-connect send → 飞书
```

### 各阶段成果

| 阶段 | 内容 | 状态 |
|------|------|------|
| Phase 0 | 探针：pipe + NDJSON 确认 | ✅ |
| Phase 1 | 非阻塞 NDJSON 解析 + 事件落盘 | ✅ |
| Phase 2 | WebSocket 服务器 (9876) | ✅ WS relay 有 bug，暂不用 |
| Phase 2.5 | bridge 轮询 JSONL → cc-connect → 飞书 | ✅ 当前方案 |

### 关键发现

- VS Code 入口：`claudeCode.claudeProcessWrapper` 设置项
- 全部 pipe 模式，NDJSON 输出
- Windows 命令行 8191 字符限制：需将 `claude.exe` 复制到 npm 全局目录，使 Go 优先匹配 .exe 绕过 .cmd
- 飞书直连国内服务器，不需代理
- 微信 ilink API 在此网络环境不可用（TCP 连接超时）

### bridge 运行方式

```powershell
$env:BRIDGE_SESSION = "feishu:oc_9837e218cd51ec1fa5f14ec230441973:ou_2f843da75285efd296cec87ed116c1b4"
C:\Users\易朝亮\.cc-connect\claude-proxy\bin\bridge.exe
```

### 已知问题

- WS relay goroutine 事件推送有 bug（客户端收不到），暂时用 JSONL 轮询替代
- proxy 端口 9876 多实例冲突（新 Claude 会话覆用旧端口时 relay 脱节）
- `claude.exe` npm 目录副本在 `npm update` 后可能丢失

## 核心约束

- 不破坏 VS Code Claude 扩展原有功能
- Go 单二进制
- bridge 轮询不阻塞主链路
