# CONTEXT.md — claude-proxy

## 项目目标

让 VS Code 里正在运行的 Claude Code 会话同步到飞书手机端。透明代理 + cc-connect 桥接。

## 目标受众

- **用户**：拾易
- **下游系统**：cc-connect（飞书），claude-bridge（JSONL 轮询 → cc-connect send）

## 当前版本

**v0.1-stable** — VS Code Claude 会话只读同步到飞书。

### 架构

```
VS Code Claude → claude-proxy (透明代理) → events/*.jsonl → claude-bridge (2s轮询) → cc-connect → 飞书
```

### 功能

| 功能 | 状态 |
|------|------|
| VS Code 透明代理，不影响现有使用 | ✅ |
| NDJSON 事件解析（assistant/user/tool_use/session_start） | ✅ |
| events JSONL 落盘（脱敏、限长） | ✅ |
| bridge 轮询 → cc-connect → 飞书消息推送 | ✅ |
| bridge 开机自启（Windows 计划任务） | ✅ |
| 消息精简（只推 assistant/user/tool_use，隐藏 delta/result） | ✅ |
| 一键启动 start-all.bat | ✅ |
| 安装/恢复脚本 | ✅ |
| WebSocket 实时推送 | ⚠️ relay bug，暂用轮询 |

### 日常使用

```text
启动：Win+R → C:\Users\易朝亮\.cc-connect\start-all.bat
飞书收发：打开飞书 App → 找到应用 → 对话
VS Code：正常使用 Claude，会话自动同步到飞书
```

### 已知限制

- bridge 轮询有 2s 延迟
- `npm update` 后需重建 claude.exe 副本
- WebSocket relay 有 bug，暂不启用

## 各阶段成果

| 阶段 | 内容 | 状态 |
|------|------|------|
| Phase 0 | 探针：pipe + NDJSON 确认 | ✅ |
| Phase 1 | 非阻塞 NDJSON 解析 + 事件落盘 | ✅ |
| Phase 2 | WebSocket 服务器 | ✅ relay bug，默认关闭 |
| Phase 2.5 | bridge JSONL 轮询 → 飞书 | ✅ |
| 稳定化 | enable_ws=false, 安装/恢复脚本, git tag, 开机自启 | ✅ |

## 下一步

| 优先级 | 工作 |
|--------|------|
| P3 | Phase 4 设计：stdin 注入（飞书控制 Claude） |
| P4 | Phase 4 PoC 开发 |
| P5 | WebSocket relay 修复 |

## 核心约束

- 不破坏 VS Code Claude 扩展原有功能
- 仅监听 127.0.0.1
- bridge 轮询不阻塞主链路
- Go 单二进制
