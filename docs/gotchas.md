# claude-proxy 踩坑记录

> 项目特定的部署、配置、运维陷阱。跨项目通用的记全局 `~/.claude/gotchas.md`。

---

## 开机自启（当前方案：启动文件夹）

- [2026-06-17] `schtasks /create /delete` 需管理员权限 → 改用启动文件夹（`shell:startup`）统一管理所有自启，不再依赖计划任务
- [2026-06-16] `start-bridge.bat` 需设 `HTTPS_PROXY`/`HTTP_PROXY` 环境变量，否则 bridge 调 `cc-connect send` 可能走不了代理
- [2026-06-17] bridge 启动时 cc-connect 未就绪导致退出码 1 → 在 `start-cc.bat` 中加 20s 延迟 + `start-bridge.bat` 再加 30s 延迟

## Windows 计划任务（已废弃）

> v0.1.2 起 bridge 自启已迁移到启动文件夹，不再依赖 schtasks。以下条目仅供历史参考。

- [2026-06-16] `schtasks /Create` 创建的计划任务默认可能 Disabled
- [2026-06-16] Git Bash 中 `schtasks /query` 被 Git 路径翻译拦截

## 配置与脚本

- [2026-06-17] `config.json` 中文路径乱码 + 反斜杠双重转义 → 根因是 `Replace('\', '\\')` + `ConvertTo-Json` 双重转义；修复：去掉手动转义，统一用 `Set-Content -Encoding UTF8`
- [2026-06-17] repair-config 读取损坏 config.json 时 `ConvertFrom-Json` 直接炸 → 加 try/catch 容错，损坏时自动重建
- [2026-06-17] PowerShell `$ErrorActionPreference = "Stop"` 下 proxy stderr 的 probe 日志被当成错误中断脚本 → `--version` 验证改用 `cmd /c` 包裹避免 NativeCommandError
- [2026-06-17] install/restore 脚本在 VS Code 内执行会杀掉当前 Claude → 加 `$env:VSCODE_PID` / `TERM_PROGRAM` 检测 + 拒绝执行 + `-Force` 参数
- [2026-06-17] `.bat` 文件含中文时 cmd.exe 编码错乱 → `rem` 注释 + `set` 命令被拆断，报"不是内部或外部命令"；解法：bat 文件全部用纯 ASCII + `%USERPROFILE%` 替代中文路径，用 `[System.Text.Encoding]::ASCII` 保存
