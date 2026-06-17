# claude-proxy

VS Code Claude 扩展的透明中间人代理。旁路捕获 NDJSON 会话流 → 事件落盘 → bridge 轮询 → cc-connect → 飞书实时同步。

## 架构

```
VS Code Claude → proxy (透明转发) → events/*.jsonl → bridge (2s轮询) → cc-connect → 飞书
```

## 目录结构

```
cmd/claude-proxy/     # 透明代理入口（替换 VS Code 的 claude 二进制）
cmd/bridge/           # 轮询 events 文件 → cc-connect send → 飞书
internal/
  config/             # 配置（防递归、BOM 兼容）
  launcher/           # 启动真实 Claude + 透传 + 旁路
  stream/             # 原始流转发 + 非阻塞 sidecar
  parser/             # NDJSON 解析 + 事件标准化 + 截断
  writer/             # 异步缓冲 JSONL 事件写入
  eventbus/           # 非阻塞事件广播（Phase 2）
  ws/                 # WebSocket 服务端（已知 relay bug）
  security/           # token 生成/脱敏/白名单 env
  logging/            # slog 封装
  probe/              # Phase 0 探针数据捕获
docs/specs/           # 设计文档
```

## 构建

```bash
cd F:\Documents\Projects\claude-proxy
go env -w GOPROXY=https://goproxy.cn,direct
go mod tidy
go build -o %USERPROFILE%\.cc-connect\claude-proxy\bin\claude.exe .\cmd\claude-proxy\
go build -o %USERPROFILE%\.cc-connect\claude-proxy\bin\claude-bridge.exe .\cmd\claude-bridge\
```

## 安装

> ⚠️ **必须在外部 PowerShell 窗口中执行**，禁止在 VS Code 集成终端或 Claude Code 会话中运行。

```powershell
cd F:\Documents\Projects\claude-proxy
powershell -ExecutionPolicy Bypass -File .\scripts\install-proxy.ps1 -Force
```

先用 `-Force` 省略做 dry-run 预览：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-proxy.ps1
```

## 一键启动

```
C:\Users\易朝亮\.cc-connect\start-all.bat
```

自动完成：清 CLAUDECODE → 设代理 → 启 cc-connect → 等 10 秒 → 启 bridge。

## 恢复原始 Claude

> ⚠️ **必须在外部 PowerShell 窗口中执行**，禁止在 VS Code 集成终端或 Claude Code 会话中运行。

```powershell
cd F:\Documents\Projects\claude-proxy
powershell -ExecutionPolicy Bypass -File .\scripts\restore-claude.ps1 -Force
```

先用 `-Force` 省略做 dry-run 预览：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\restore-claude.ps1
```

proxy 无需手动启动，通过 VS Code `claudeCode.claudeProcessWrapper` 自动拦截。

## 故障排查

### VS Code 报 "Claude Code process exited with code 1"

这是 VS Code Claude Code 扩展更新后，`real_bin` 指向旧版本目录导致的（如 `2.1.178` → `2.1.179`）。

**修复**：运行 `repair-config.ps1` 自动扫描最新扩展版本并更新配置，无需重建 proxy。如果 `config.json` 已损坏（乱码/非法 JSON），该脚本会自动重建配置文件。

```powershell
cd F:\Documents\Projects\claude-proxy
powershell -ExecutionPolicy Bypass -File scripts\repair-config.ps1
```

修完后 `Ctrl+Shift+P -> Reload Window` 即可。

如果 repair 也失败，运行完整安装：
```powershell
powershell -ExecutionPolicy Bypass -File scripts\install-proxy.ps1
```

## 注意事项

- 代理内部绝不通过 PATH 查找 `claude`
- Windows 需将 `claude.exe` 复制到 npm 全局目录规避命令行限制
- 仅监听 127.0.0.1
- bridge 轮询不阻塞主链路
- 项目背景详见 CONTEXT.md
