# claude-proxy 踩坑记录

> 项目特定的部署、配置、运维陷阱。跨项目通用的记全局 `~/.claude/gotchas.md`。

---

## Windows 计划任务

- [2026-06-16] `schtasks /Create` 创建的计划任务默认可能 Disabled → 创建后必须 `schtasks /change /tn "任务名" /enable` 启用，否则登录时不触发
- [2026-06-16] Git Bash 中 `schtasks /query` 的 `/query` 被 Git 路径翻译拦截 → 报"无效参数"，必须用 PowerShell 执行 schtasks 命令

## 开机自启

- [2026-06-16] bridge 自启依赖计划任务 + 启动文件夹两个入口，任一禁用都会断链 → 排查时先 `schtasks /query` 看任务状态，再检查 `shell:startup` 文件夹
- [2026-06-17] `schtasks /create /delete` 需管理员权限，普通 PowerShell 跑 Access Denied → 改用启动文件夹（`shell:startup`）统一管理所有自启，不需要管理员
- [2026-06-16] `start-bridge.bat` 需设 `HTTPS_PROXY`/`HTTP_PROXY` 环境变量，否则 bridge 调 `cc-connect send` 可能走不了代理
