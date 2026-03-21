# Task Plan

## Current Execution Snapshot
- **Current goal:** Task 11: 把 UDP ingest 和只读 admin APIs 接到真实后端仓储上，形成 syslog -> parser -> employee match -> attendance aggregate -> pending report persist 的最小闭环。
- **Execution status:** in progress
- **Final checkpoints:**
  - bootstrap 负责加载 config、打开 MySQL、执行 migration，并提供可关闭的 app 依赖
  - 新增清晰的 `SyslogPipeline` 服务入口，且 parse 成功/失败路径都被测试锁定
  - UDP listener、parser、employee match、attendance aggregate、pending report 通过真实仓储串起来
  - GET `/api/employees` `/api/logs` `/api/attendance` `/api/settings` 读取真实仓储而不是空数组
  - `main.go` 完成真实启动接线并在 shutdown 时关闭资源
  - 先写失败测试，再实现最小代码，并完成 `go test` 验证

## Goal
完成 Task 11 的后端主链路接线，保持 KISS/YAGNI，不引入前端真 API、外部 HTTP 发送或管理员写操作。

## Phases
| Phase | Status | Description |
|---|---|---|
| 1 | complete | 初始化审查计划文件并记录范围 |
| 2 | complete | 读取目标文件与提交内容，逐条核对规格 |
| 3 | complete | 汇总结论并给出最终审查结果 |
| 4 | complete | 复审 Task 2 修复提交 `1d1e5f518746c62c9cea599de31da601ea4ba183` 并核对上次问题是否关闭 |
| 5 | complete | 按生产就绪标准独立复审 Task 2 提交，补充编译验证与设计对照 |
| 6 | complete | 复审 Task 3 提交 `312f679440296890e7c77fff6fcf2456fc1ad24c`，核对 parser 约束、领域契约与测试覆盖 |
| 7 | complete | 复审 Task 3 修复提交 `855caa7`，确认时间字段、unsupported verb 和测试覆盖问题是否已关闭 |
| 8 | complete | 复审 Task 4 提交 `02cdcd4bf4c43ad1b92294cce3002561d1312701`，核对 attendance processor、report service、repository skeleton 与测试覆盖 |
| 9 | complete | 复审 Task 4 修复提交 `97a6cfb0fcd85879f40b385d0617bf0c52e1c6d2`，重点核对幂等键稳定性、`LastCalculatedAt` 歧义与范围控制 |
| 10 | complete | 复审 Task 5 提交 `85f5392bae1b9560d6aaebc0e5e6ec56992b5d69`，核对日终服务、UDP listener、scheduler skeleton 与 `main.go` 接线；结论为不符合规格 |
| 11 | complete | 复审 Task 5 修复提交 `299996714d84d2866e3c49083d4b795db4acc63c`，重点确认 bootstrap/config 接线与 `LastCalculatedAt` 超范围行为已移除 |
| 12 | complete | 按用户给定范围 `97a6cfb0fcd85879f40b385d0617bf0c52e1c6d2..299996714d84d2866e3c49083d4b795db4acc63c` 重新执行生产就绪代码审查并输出结构化结论 |
| 13 | complete | 复审 Task 6 提交 `f87d5b2914c6f1e7048395b81535de989549b14f`，核对 Admin HTTP API stub 路由、占位响应与测试是否满足最小规格 |
| 14 | complete | 按用户指定范围 `299996714d84d2866e3c49083d4b795db4acc63c..f87d5b2914c6f1e7048395b81535de989549b14f` 重新执行生产就绪代码审查，补查 HTTP 实际暴露路径与启动接线 |
| 15 | complete | 按用户指定新 head `19eb2ca287cb8ac0942925750be9681eba050053` 重新独立复审 Task 6，重点核对运行态 HTTP 暴露、router 接线与测试锁定力度 |
| 16 | complete | 复审 Task 7 提交 `178427b0b19873d62329116271f45f2e5c7b78ac`，核对 React shell、5 个页面路由、控制台风格、占位布局与测试是否满足最小规格 |
| 17 | complete | 复审 Task 7 修复提交 `150a0b48a29758fb0c36abcc1d75203d5727a234`，重点核对真实 router 测试、`AppShell` fallback 移除、字体依赖移除、lockfile 提交与范围控制 |
| 18 | complete | 复审 Task 8 指定前端页面流实现，核对 mock API、3 个表单/表格组件、3 个页面本地 CRUD 流程、异常行“人工修正”动作，以及 `attendance-page.test.tsx` 是否锁定目标行为 |
| 19 | complete | 按新 head `30fc9561973322709b95d628a57603b206c21912` 复审 Task 8 修复情况，重点确认员工页 edit/disable mock flow、占位文案移除，以及 `attendance-page.test.tsx` 是否只在异常行上锁定 `人工修正` 动作 |
| 20 | complete | 按用户指定范围 `150a0b48a29758fb0c36abcc1d75203d5727a234..30fc9561973322709b95d628a57603b206c21912` 执行 Task 8 生产就绪代码审查，重点核对职责边界、文件结构、页面本地数据流与测试约束力度 |
| 21 | complete | 仅复核最新提交 `ef32ed9c0e17356a6a7bf74d8c3c789e6ca1e099` 对 Task 8 质量问题的修复，重点检查测试补强、输入校验、稳定 key，以及是否仍有阻塞收口问题 |
| 22 | complete | 仅复核最新提交 `15b9150f5442434cf3d6b8ba1739f16254284c40` 的 Task 9 改动，重点检查集成测试闭环、docker-compose 三服务约定、README 实现边界描述，以及 `send-sample-syslog.sh` 可用性 |
| 23 | complete | Task 10：接入真实 MySQL bootstrap、配置与 repository 持久化基础，先写失败测试再实现最小代码 |
| 24 | complete | 复核提交 `2591b514e9337b6dc3bc9f6b1aedcdd979a85a91` 的 Task 10 规格符合性，聚焦 MySQL 配置、bootstrap、repository 覆盖与设计冲突 |
| 25 | complete | 修正 Task 10 review 指出的三项问题：固定时区、`MYSQL_DSN` 优先、migration 幂等，并补一个轻量 `bootstrap.App.DB` 承载位 |
| 26 | complete | 复核当前 head（含 `2591b51` 和 `5f8eb75`）的 Task 10 MySQL 持久化基础实现，聚焦 repository 设计、`database/sql` 风险、bootstrap 边界与测试锁定力度 |
| 27 | complete | 继续修正 Task 10 持久化基础：raw `MYSQL_DSN` 规范化、report NULL 文本列读取、`LastInsertId` 负数路径收口 |
| 28 | complete | 仅复核最新修复提交 `cea6200e20dd8040da84a5a43f75eecd03d9b246` 是否关闭 Task 10 的 DSN/NULL 文本列/测试覆盖问题 |
| 29 | complete | 仅复核最新修复提交 `c8670979908d4255bae67c094c72d7dd362ab5c5` 是否关闭 split-field `MYSQL_PARAMS` 冲突键过滤问题并由测试锁定 |
| 30 | complete | Task 11：把 UDP ingest 和只读 admin APIs 接到真实后端仓储上，形成最小可跑闭环 |

## Review Checklist
- 核对 8 个指定文件是否存在
- 检查文件内容是否满足最小要求
- 检查是否有未请求的额外功能/文件
- 如有必要，检查提交内容是否与声明一致
- 对 Task 2 复审时，重点核对 report history、MAC 映射、attendance state、employees/syslog_messages、system_settings seed rows
- 对 Task 3 复审时，重点核对 parser 是否满足最小领域契约、是否误判事件类型、测试是否覆盖核心分支
- 对 Task 3 修复复审时，重点核对 `EventDate`/`EventTime` 赋值、unsupported verb 错误返回，以及 4 个目标测试场景
- 对 Task 4 复审时，重点核对首连/末断开逻辑、report 幂等键构造、pending report 创建、repository 是否保持最小骨架，以及测试是否覆盖 3 个必测场景
- 对 Task 5 复审时，重点核对 `FinalizeForDay` 两个分支、UDP listener 的最小接口、scheduler 骨架、`main.go` 是否复用现有 bootstrap/config 且未扩 scope
- 对 Task 6 复审时，除文件/测试清单外，还要确认“Expose”是否真的经由进程入口暴露为可访问 HTTP surface，而不只是包内未接线的 router
- 对 Task 6 二次复审时，额外要求运行进程本身验证 `/api/attendance` 可达，并区分“测试锁定 NewServer 使用 router”与“是否直接锁定 main.go 接线”
- 对 Task 7 复审时，重点核对 5 个导航项与路由是否齐全、页面标题和占位结构是否有真实控制台感、测试是否真正锁定 shell 导航渲染而非仅验证静态文本
- 对 Task 7 二次复审时，必须直接验证测试是否挂载真实 router、至少覆盖一个非默认路由、`AppShell` 是否仅保留生产路径、样式是否不再依赖 Google Fonts 运行时导入，以及 lockfile 是否已被 git 跟踪
- 对 Task 8 复审时，重点核对 `frontend/src/lib/api.ts` 是否只提供小型异步 mock helper；3 个页面是否保持本地/mock CRUD-like flow；`AttendancePage` 是否明确渲染异常行并提供可见的 `人工修正` 操作；实现是否未引入真实请求、TanStack Query 或额外状态库；`attendance-page.test.tsx` 是否真实锁定该页面行为
- 对 Task 8 二次复审时，额外要求确认 `EmployeesPage` 是否已有最小本地 edit/disable 动作而非 create-only，任何 “待接入/placeholder” 文案是否已移除，以及 `attendance-page.test.tsx` 是否把 `人工修正` 动作与 exception row 绑定验证
- 对 Task 11 实现时，重点核对 pipeline 的失败/成功分支、只读 admin API 的真实 JSON、`main.go` 的 shutdown 接线、以及是否保持在当前明确范围内

## Errors Encountered
| Error | Attempt | Resolution |
|---|---|---|
| Serena 无法解析 Go 符号（项目语言配置仅识别 TypeScript） | 1 | 回退到定向 shell 只读检查目标 Go 文件与测试输出 |
| `session-catchup.py` 默认插件路径不存在 | 1 | 改用技能实际安装路径 `/Users/wesleyw/.codex/skills/planning-with-files/skills/planning-with-files/scripts/session-catchup.py` |
