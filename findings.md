# Findings

## Task 11 Initial Context Notes
- 当前 `main.go` 只创建了空的 `Dependencies{}`，`UDPListener` 的 handler 也只是空实现，因此 syslog 入口和 admin HTTP 入口都还没有接到真实仓储。
- HTTP 只读 handler 目前统一返回空 `items`，需要先补最小查询注入，再把路由层接到真实 repo/service。
- 现有 repository 已经足够支撑 Task 11 的最小闭环，不需要扩展到写 admin API、外部 HTTP 发送或新的数据抽象。
- `AttendanceProcessor` 和 `ReportService` 已经具备本任务需要的领域行为，Task 11 重点是把它们编排到一个明确的 pipeline 入口。

## Task 11 Implementation Notes
- 新增 `service.SyslogPipeline` 作为唯一的 syslog ingest 服务入口，负责 raw message 保存、AP syslog parse、employee match、attendance aggregate 和 clock-in pending report 去重保存。
- `main.go` 现在通过 `bootstrap.New` 拿到真实 app 依赖，再把 UDP listener 和 admin HTTP server 接到真实仓储上；shutdown 时会关闭 listener、HTTP server 和 DB。
- admin 只读路由已经改成从真实仓储读取数据，`/api/logs` 会把 recent message 和 recent parsed event 合并返回。
- 当前测试已经覆盖 pipeline 的 parse success / parse failure，以及至少两个真实 JSON 路由的输出内容。
- 最终验证命令已经通过：`cd backend && go test ./internal/service ./internal/http/handlers -v` 与 `cd backend && go test ./...`。

## Task 10 Final Fix-Only Review Notes (`c8670979908d4255bae67c094c72d7dd362ab5c5`)
- 本次提交只改了 `backend/internal/bootstrap/mysql.go` 和 `backend/internal/bootstrap/mysql_test.go`，范围准确聚焦在上轮剩余问题。
- split-field 路径现在会在两个位置过滤受控键：构建 `Params` 时跳过 `charset/parseTime/multiStatements/loc`，规范化阶段也会再次从 `cfg.Params` 中删除这些键，因此冲突 `MYSQL_PARAMS` 不会再被 `FormatDSN()` 追加回最终连接串。
- 新增测试 `TestOpenMySQLSplitFieldsIgnoreConflictingControlledParams` 明确断言 `parseTime=false`、`loc=UTC`、`multiStatements=false` 不得出现在最终 DSN，同时保留非受控参数 `foo=bar`，足以锁定这次修复目标。
- `cd backend && go test ./internal/bootstrap -v` 通过。

## Task 10 Fix-Only Review Notes (`cea6200e20dd8040da84a5a43f75eecd03d9b246`)
- 本次修复提交范围很聚焦，只修改了 `bootstrap/mysql.go`、`bootstrap/mysql_test.go`、`report_repository.go`、`mysql_helpers.go` 和 `mysql_repository_test.go` 5 个文件。
- `attendance_reports` 的 `payload_json` / `response_body` 读取风险已经关闭：repository 读路径改成 `sql.NullString` 扫描，再通过 helper 回落为空字符串；新增测试也覆盖了 `NULL` 读回场景。
- `parseInsertedID` 的负数 `LastInsertId` 路径也已经收口，并补了对应单测。
- 但 DSN 规范化仍未完全关闭：`buildMySQLDSN` 在 split-field 路径会把 `parseTime` / `loc` / `multiStatements` 继续保留到 `mysql.Config.Params`，而 `normalizeMySQLConfig` 只是设置了结构体字段，没有把这些保留键从 `Params` 清掉。
- 对 `go-sql-driver/mysql v1.9.3`，`FormatDSN()` 会先输出结构体字段中的 `loc=Asia/Shanghai&multiStatements=true&parseTime=true`，再把 `cfg.Params` 全量追加到 query string。最小复现实验显示，只要 `MYSQL_PARAMS` 带冲突值，最终 DSN 会变成：`...?loc=Asia%2FShanghai&multiStatements=true&parseTime=true&charset=utf8mb4&loc=UTC&multiStatements=false&parseTime=false`。
- 因此，第 1 点只修好了 raw `MYSQL_DSN` 路径，没有把 split-field config 路径也完全统一到同一个连接契约；同时测试也没有新增“冲突 `MYSQL_PARAMS` 不得泄漏/覆盖”的断言，所以还不能据此放行整个 Task 10。

## Task 10 Final Review-Fix Closure Notes
- `OpenMySQL` now normalizes both raw DSN and split-field inputs through one contract: `parseTime=true`, `multiStatements=true`, and `loc=Asia/Shanghai` are always enforced.
- `attendance_reports` read paths now scan `payload_json` and `response_body` through `sql.NullString`, so `NULL` values no longer explode during read queries.
- `parseInsertedID` now returns an error for negative `LastInsertId`, making the helper less permissive without adding a wider error-handling framework.

## Task 10 Head Re-Review Notes (`2591b51` + `5f8eb75`)
- 审查范围锁定在 `15b9150f5442434cf3d6b8ba1739f16254284c40..5f8eb750bf57b37f4b8d21f364ff310c2299ef56` 的后端 MySQL 持久化基础实现，不扩展到前端或 UDP/HTTP 业务接线。
- repository 结构整体保持按聚合拆分，没有出现 god object；`mysql_helpers.go` 统一承载 limit/null/inserted-id 小工具，方向上符合 SRP/KISS。
- 当前最明显的 bootstrap 边界风险在 `OpenMySQL`：当 `MYSQL_DSN` 被设置时，代码直接原样透传 DSN，不再保证 `parseTime=true`、`multiStatements=true` 和 `loc=Asia/Shanghai`。但 repository 的 `time.Time/sql.NullTime` 扫描和 `RunMigrations` 的整份多语句 SQL 执行都隐含依赖这些参数，因此 raw DSN 路径并不自洽。
- `attendance_reports` 表把 `payload_json`、`response_body` 定义为 `NULL` 列，但 repository 读路径直接扫描到 `string` 字段；只要库里出现 `NULL`，`Scan` 就会在运行时失败。当前测试只覆盖了非空字符串场景，没锁住这个风险。
- 测试层主要是 happy path：config 只测默认值/覆盖；bootstrap 只测 DSN 直通、拼接和 migration 调用；repository 只测成功查询/保存。未覆盖 `sql.ErrNoRows`、NULL 列扫描、`LastInsertId` 异常、raw DSN 缺失必要参数等风险路径。
- `cd backend && go test ./internal/config ./internal/bootstrap ./internal/repository` 与 `cd backend && go test ./...` 在当前树均通过；问题集中在运行时契约和测试约束，而不是编译失败。

## Task 10 Review-Fix Closure Notes
- `TIMEZONE` no longer influences runtime config; `LoadConfigFromEnv` always returns `Asia/Shanghai`.
- `MYSQL_DSN` is now a first-class override for `OpenMySQL`; split host/port/user/password/database/params remain available for compose defaults.
- Split DSN defaults now keep `loc=Asia/Shanghai`, and the bootstrap helper also defaults to `Asia/Shanghai` when it synthesizes a DSN.
- `RunMigrations` is now safe to re-run because the embedded SQL is idempotent: every table uses `CREATE TABLE IF NOT EXISTS`, and the seed rows use `INSERT IGNORE`.
- I added a light `bootstrap.App.DB` field so the app struct can carry a DB handle later without pulling `main.go` into this fix.

## Task 10 Review Notes (`2591b514e9337b6dc3bc9f6b1aedcdd979a85a91`)
- 审查范围仅限本次提交新增的 MySQL 配置、bootstrap、migration 入口和 concrete repositories；`main.go`、HTTP、ingest、frontend 未扩 scope。
- repository 覆盖面本身基本齐全：员工按 MAC 查找与列表、syslog message 保存/最近列表、client event 保存/最近列表、attendance record 查找/保存/日期范围列表、attendance report 按 idempotency 查找/保存/按 record 列表、system settings 读取/列表都已落地。
- 没有引入 GORM；Go 依赖只新增 `go-sql-driver/mysql` 和 `go-sqlmock`。
- 但本次提交与已批准设计仍有两处关键冲突：
  - 时区契约被削弱：配置允许 `TIMEZONE` 覆盖 `Asia/Shanghai`，而 MySQL DSN 默认又写死 `loc=Local`，没有绑定到 `cfg.Timezone`。在当前 `docker-compose.yml` 下 backend 容器也没有显式设置时区，因此持久化时间语义会跟随运行环境本地时区，而不是设计要求的固定 `Asia/Shanghai`。
  - 配置层没有真正支持单一 MySQL DSN 输入；当前只有 host/port/user/password/database/params 分段字段，由 bootstrap 内部拼装 DSN。这与“配置支持 MySQL DSN”的目标不完全一致，也让下一步接线仍需要知道这套拆分约定。
- `bootstrap.OpenMySQL` 和 `bootstrap.RunMigrations` 作为独立 helper 可被后续接线直接调用，但它们没有并入 `bootstrap.App` 生命周期；下一步仍需要新增 DB ownership / close / repository 组装逻辑。
- `RunMigrations` 当前每次直接执行整份 `001_init.sql`。由于 SQL 中使用裸 `CREATE TABLE` 和普通 `INSERT` seed，这个入口对重复启动并不幂等，更适合一次性初始化而不是无条件放进启动路径。
- 针对当前树运行 `cd backend && go test ./internal/config ./internal/bootstrap ./internal/repository` 已通过；问题集中在规格契约，而不是编译或单测失败。

## Task 10 Discovery Notes
- 当前 `backend` 仅有两个 repository 骨架接口文件，且没有任何 concrete MySQL 实现；这正好符合 Task 10 需要补齐持久化层的范围。
- `main.go` 目前只依赖 `bootstrap.New(os.Getenv)` 提供的配置，未接入数据库对象，因此 Task 10 可以把 DB 打开和 migration 入口放在 `bootstrap` 包里，不必触碰 `main.go`。
- 现有 domain 已经覆盖 `Employee`、`EmployeeDevice`、`SyslogMessage`、`ClientEvent`、`AttendanceRecord`、`AttendanceReport`、`SystemSetting`，与 migration 字段基本对齐，适合直接写 repository 映射测试。
- HTTP handlers 现在仍是空壳返回，这意味着 repository 接口可以优先服务后续 ingest/只读 API，而不用为了当前 handler 过度设计。
- 这次实现会优先采用按实体拆分的 repository，而不是单一的万能仓库，保持职责单一并控制文件体量。
- 测试层会优先用 `sqlmock` 之类的轻量依赖验证 SQL 语义，避免引入 ORM 或真实 MySQL 容器做单元测试。
- 首轮 `go test ./internal/config ./internal/repository ./internal/bootstrap -v` 已经红了，错误符合预期：`Config` 尚未扩展 MySQL 字段，且模块还缺少 `github.com/DATA-DOG/go-sqlmock` 依赖。
- 最终实现维持了最小边界：config 只提供本地 compose 友好的 MySQL 连接参数，bootstrap 只负责打开 DB 与执行一次性 migration，repository 按实体拆分，没有引入 ORM、事务层或 HTTP 接线。

## Final Execution Notes
- Task 8 最终收口提交为 `ef32ed9c0e17356a6a7bf74d8c3c789e6ca1e099`，已补齐人工修正状态迁移测试、员工与设置页 happy-path 测试、表单校验，以及稳定 `employee.id` key。
- Task 9 最终收口提交为 `15b9150f5442434cf3d6b8ba1739f16254284c40`，已补齐：
  - `docker-compose.yml` 的 `mysql` / `backend` / `frontend` 三服务本地联调栈
  - `backend/tests/integration/syslog_flow_test.go` 服务级闭环集成测试
  - `scripts/send-sample-syslog.sh` 样例 UDP syslog 发送脚本
  - `README.md` 的启动、测试、发包与边界说明
- 当前树验证结果：
  - `cd backend && go test ./tests/integration -run TestSyslogFlow -v` 通过
  - `cd backend && go test ./...` 通过
  - `cd frontend && npm test` 通过
  - `cd frontend && npm run build` 通过
  - `docker compose config` 通过
- 质量复核结论：
  - Task 8：Ready to merge `Yes`
  - Task 9：Ready to merge `Yes`

## Task 9 Fix-Only Review Scope
- Review target commit: `15b9150f5442434cf3d6b8ba1739f16254284c40`
- Review intent: only assess Task 9 changes in the latest commit.
- Required focus:
  - integration test 是否真实验证 `parser -> processor -> day-end -> report service` 闭环
  - `docker-compose.yml` 是否满足 mysql/backend/frontend 三服务和端口映射约定
  - `README.md` 是否准确描述当前实现边界，避免过度承诺
  - `send-sample-syslog.sh` 是否可用，参数与默认值是否合理

## Task 9 Fix-Only Review Notes
- `git show --stat 15b9150f5442434cf3d6b8ba1739f16254284c40` 显示本次改动只涉及 4 个文件，范围集中在 Task 9 所需的联调与文档支撑物。
- `backend/tests/integration/syslog_flow_test.go` 不是空壳：它实际调用 `parser.ParseAPSyslog`、`AttendanceProcessor.ApplyEvent`、`DayEndService.FinalizeForDay` 和 `ReportService.CreatePendingReport`，并对解析结果、首连/末断、day-end 状态归一、report type/status/idempotency key 与 payload timestamp 做断言，已形成服务级闭环验证。
- 该集成测试仍是“服务级闭环”，没有穿过 UDP listener、main wiring、repository 或 MySQL；但 `README.md` 也明确把它表述为“服务级闭环集成测试”，没有夸大成端到端链路。
- `docker-compose.yml` 经过 `docker compose config --services` 校验，确实定义了 `mysql`、`backend`、`frontend` 三个服务；端口映射分别为 `3306:3306`、`8080:8080`、`5514:1514/udp`、`5173:5173`，与 README 描述一致。
- `README.md` 当前对实现边界的描述基本准确：明确说明了 UDP handler 仍是占位、样例 syslog 发送主要用于本地联调/端口验证、MySQL 尚未进入测试闭环、前端仍是 mock API。
- `scripts/send-sample-syslog.sh` 经 `bash -n` 语法检查通过，并通过本地临时 UDP 接收端实际验证：默认参数模型可用，`both` 会发出 connect + disconnect 两条消息；非法模式会返回 usage。
- 脚本默认目标 `127.0.0.1:5514/udp` 与 compose 宿主机映射一致，`connect|disconnect|both` 三种模式和 `SYSLOG_HOST` / `SYSLOG_PORT` 覆盖方式都合理。
- 本轮没有发现阻塞 Task 9 收口的重要问题。唯一轻微缺口是脚本依赖宿主机存在 `nc`，README 没有显式写出该前提；这不影响当前提交的核心可用性，但可以补充说明。

## Task 8 Fix-Only Re-Review Scope
- Review target commit: `ef32ed9c0e17356a6a7bf74d8c3c789e6ca1e099`
- Review intent: only verify whether the latest commit closes the previously reported Task 8 quality issues.
- Required focus:
  - 考勤页交互测试是否覆盖人工修正后的状态迁移
  - 员工页 create/update/disable 与设置页 save 是否补上最小 happy-path 测试
  - 纯空白员工字段与非法数字设置是否被拒绝
  - `EmployeesPage` 是否改成稳定 key
  - 是否还残留阻塞 Task 8 收口的重要问题

## Task 8 Fix-Only Re-Review Notes
- `git show --stat ef32ed9c0e17356a6a7bf74d8c3c789e6ca1e099` 显示本次修复只涉及 7 个前端文件，范围聚焦在上轮指出的 Task 8 质量问题。
- `frontend/src/test/attendance-page.test.tsx` 现在在异常行点击 `人工修正` 后，进一步断言该行出现 `已修正`、页面出现提交成功消息、且按钮消失，已经覆盖从 exception 到 corrected 的最小状态迁移。
- `frontend/src/test/employees-page.test.tsx` 新增 happy-path 测试，串起 create -> edit -> disable 三个 mock flow；另有一条测试验证纯空白员工字段被拒绝。
- `frontend/src/test/settings-page.test.tsx` 新增 happy-path save 测试，并验证非法数字设置会被拒绝。
- `frontend/src/features/employees/components/EmployeeForm.tsx` 在提交前先对 `name/team/badge` 做 `trim()` 并拒绝空串；`frontend/src/features/settings/components/SettingsForm.tsx` 在提交前对三个数值字段做整数与最小值校验。
- `frontend/src/features/employees/EmployeesPage.tsx` 已将列表 key 从可编辑的 `employee.badge` 改为稳定的 `employee.id`。
- `frontend/src/lib/api.ts` 新增 `resetMockData()`，用于在测试前重置本地 mock store，避免测试间共享状态污染。
- 验证结果：`cd frontend && npm test` 与 `cd frontend && npm run build` 在当前树通过。
- 本轮没有再发现阻塞 Task 8 收口的重要问题；剩余只是不影响合并的轻微事项，例如 mock API 本身仍未在函数层做防御式校验，但在当前“仅前端本地/mock 流程”范围内已非阻塞项。

## Task 8 Production Review Scope
- Review target range: `150a0b48a29758fb0c36abcc1d75203d5727a234..30fc9561973322709b95d628a57603b206c21912`
- User-requested commands:
  - `git diff --stat 150a0b48a29758fb0c36abcc1d75203d5727a234..30fc9561973322709b95d628a57603b206c21912`
  - `git diff 150a0b48a29758fb0c36abcc1d75203d5727a234..30fc9561973322709b95d628a57603b206c21912`
- Review focus:
  - `frontend/src/lib/api.ts` 是否保持小型异步 mock API
  - `EmployeeForm`、`AttendanceTable`、`SettingsForm` 是否职责单一
  - `EmployeesPage`、`AttendancePage`、`SettingsPage` 是否仅接本地/mock 数据流
  - `AttendancePage` 是否在 exception row 上显示 `人工修正`
  - 文件结构、体量、职责是否符合 Task 8 计划
  - 测试是否足以锁定“先红后绿”后应保留的关键行为

## Task 8 Production Review Notes
- `git diff --stat 150a0b48a29758fb0c36abcc1d75203d5727a234..30fc9561973322709b95d628a57603b206c21912` 显示范围严格落在计划内的 8 个前端文件，没有扩散到后端或额外基础设施。
- `frontend/src/lib/api.ts` 维持了本地内存数据 + `setTimeout` 的 async mock helper 形态；检索 `frontend/src` 未发现 `fetch`、`axios`、TanStack Query 或额外状态库。
- 三个新组件 `EmployeeForm`、`AttendanceTable`、`SettingsForm` 基本都承担单一视图职责；页面组件负责加载与提交流程，文件结构与计划一致。
- `AttendancePage` 确实在 exception row 上渲染 `人工修正` 按钮；`attendance-page.test.tsx` 也按 `Arjun Patel` / `Lena Wu` 两个具体行做了可见性断言。
- `cd frontend && npm test`、`cd frontend && npm run build` 在当前树通过。
- 主要风险集中在两类：
  - 测试覆盖不足：新增的员工 CRUD-like flow、设置保存流没有测试；考勤测试只验证按钮出现在异常行，并未点击按钮确认“人工修正”后的状态迁移。
  - 表单缺少有效性守卫：员工表单允许全空白字符串经 `trim()` 后进入 mock store；设置表单会把清空输入后的 `""` 直接转成 `0` 并保存，可能产生无效本地状态。
- 文件体量方面，`frontend/src/lib/api.ts`（202 行）和 `frontend/src/features/employees/EmployeesPage.tsx`（206 行）已接近“一个文件承担过多变化点”的边界，但在当前 Task 8 范围内仍可接受，建议下个迭代开始按领域拆分。

## Task 8 Review Scope
- Review target: current working tree under `/Users/wesleyw/Project/syslog/frontend`
- Files under inspection:
  - `frontend/src/lib/api.ts`
  - `frontend/src/features/employees/EmployeesPage.tsx`
  - `frontend/src/features/employees/components/EmployeeForm.tsx`
  - `frontend/src/features/attendance/AttendancePage.tsx`
  - `frontend/src/features/attendance/components/AttendanceTable.tsx`
  - `frontend/src/features/settings/SettingsPage.tsx`
  - `frontend/src/features/settings/components/SettingsForm.tsx`
  - `frontend/src/test/attendance-page.test.tsx`
- Review focus:
  - mock API helper 是否保持小型异步本地实现
  - EmployeeForm、AttendanceTable、SettingsForm 是否存在并承担单一职责
  - EmployeesPage、AttendancePage、SettingsPage 是否都接到本地/mock CRUD-like flow
  - AttendancePage 是否对异常行提供可见 `人工修正` 动作
  - 是否引入真实后端请求、TanStack Query 或额外状态库
- `attendance-page.test.tsx` 是否锁定目标行为而非宽松占位测试

## Task 8 Review Notes
- `frontend/src/lib/api.ts` 保持为本地内存数据 + `setTimeout` 包装的异步 mock helper，没有发现真实网络请求、TanStack Query 或额外状态库。
- `EmployeeForm`、`AttendanceTable`、`SettingsForm` 三个组件都已新增，职责边界基本清晰，分别只处理表单输入或表格呈现。
- `AttendancePage` 已通过 `listAttendanceRecords` 与 `correctAttendanceRecord` 接到本地 mock 流程；异常记录行会显示 `人工修正` 按钮，普通/已修正行显示只读文案。
- `SettingsPage` 已通过 `getSettings` / `saveSettings` 接到本地 mock 读写流程。
- `EmployeesPage` 目前只实现了 `listEmployees` + `createEmployee`；页面文案仍明确标注“编辑待接入”“停用待接入”和“待后续接入编辑/停用流程”，说明员工侧的 mock CRUD-like flow 尚未完整接线。
- `frontend/src/test/attendance-page.test.tsx` 当前只断言页面上存在一次 `人工修正` 文本，没有验证该动作属于异常行，也没有锁定异常记录本身，因此测试强度不足以覆盖规格中的关键页面行为。

## Task 8 Re-Review Scope (Head `30fc9561973322709b95d628a57603b206c21912`)
- Review target: current `HEAD` and target files under `frontend/`
- Re-review focus:
  - `EmployeesPage` 是否已经接上最小 local/mock edit 与 disable 流程
  - 之前遗留的 “编辑待接入 / 停用待接入 / 待后续接入编辑/停用流程” 文案是否已经消失或被真实动作替代
  - `attendance-page.test.tsx` 是否明确验证 exception row 上可见 `人工修正`，而不是只在页面任意位置找文本

## Task 8 Re-Review Notes
- `frontend/src/lib/api.ts` 新增 `updateEmployee` 与 `disableEmployee`，都维持在内存数组上的异步 mock helper 范围内，没有引入真实请求或额外状态管理。
- `frontend/src/features/employees/EmployeesPage.tsx` 现在为每个员工卡片接入真实的本地 `编辑` 与 `停用` 动作；编辑态会复用 `EmployeeForm`，停用态会通过 mock API 将状态切到 `Disabled`，并更新本地列表。
- 针对上一轮指出的占位问题，目标文件中已不再出现 “编辑待接入”“停用待接入”“待后续接入编辑/停用流程” 等遗留文案。
- `frontend/src/features/attendance/components/AttendanceTable.tsx` 为每一行补上 `role="group"` 与稳定 `aria-label`，使测试可以按具体行约束动作。
- `frontend/src/test/attendance-page.test.tsx` 现在会在 `Arjun Patel` 行内断言存在 `人工修正` 按钮，并在 `Lena Wu` 的正常行内断言不存在该按钮；这比之前的全页文本查找更准确地锁定了“异常行上才有人工修正动作”的规格。
- `cd frontend && npm test -- --run src/test/attendance-page.test.tsx` 在当前树通过。

## Scope
- Review target: `186d066dcbc0f091f1324c540ccd087bc6d10371`
- Files under inspection:
  - `backend/go.mod`
  - `backend/cmd/server/main.go`
  - `frontend/package.json`
  - `frontend/vite.config.ts`
  - `frontend/tsconfig.json`
  - `docker-compose.yml`
  - `.env.example`
  - `README.md`

## Notes
- Commit `186d066dcbc0f091f1324c540ccd087bc6d10371` adds exactly the 8 requested files and no others.
- `backend/cmd/server/main.go` contains only `package main` and an empty `main()`.
- `frontend/package.json` includes required `name`, `private`, and `dev`/`build`/`test` scripts.
- `docker-compose.yml` defines a minimal MySQL service; no unrelated services detected.
- Current existence check `test -f backend/go.mod && test -f frontend/package.json && test -f docker-compose.yml` passes.
- Parent commit `b08fa57b33534e0491113e4ac7d0a39bdbc2c766` does not contain `backend/go.mod`, `frontend/package.json`, or `docker-compose.yml`, consistent with the expected initial failure state.
- No file appears oversized or overloaded; each file stays focused on a single bootstrap concern.
- The required pre-creation failing existence check is not directly observable in the diff; compliance is inferred from the base commit missing the checked files and the head commit containing them.

## Task 2 Review Notes
- Commit `22158a454215654a5efaa4470016629dde512115` adds exactly the 7 requested backend files and no obvious extra files.
- `backend/internal/config/config.go` and `backend/internal/config/config_test.go` implement and verify only the requested defaults: `Timezone=Asia/Shanghai` and `SyslogRetentionDays=30`.
- `cd backend && go test ./internal/config -run TestLoadConfigDefaults -v` passes in the current tree; the pre-implementation failing run is not verifiable from the final diff alone.
- The approved design document expects MAC-specific schema fields such as `employee_devices.mac_address`, `client_events.station_mac`, and report-history fields such as `attendance_reports.idempotency_key`, `payload_json`, `response_code`, `response_body`, `retry_count`.
- The submitted schema/domain skeletons instead use generic fields like `device_identifier`, `payload`, `status`, `summary`, and omit the report-history/idempotency fields required by the approved design.
- `system_settings` in the design expects an `id` plus specific bootstrap settings (`day_end_time`, `syslog_retention_days`, `report_target_url`, `report_timeout_seconds`, `report_retry_limit`); the migration only creates a key/value table and does not seed the required initial settings.

## Task 2 Re-Review Notes
- Commit `1d1e5f518746c62c9cea599de31da601ea4ba183` updates only the four schema/domain files that were previously flagged.
- `employees` now includes `system_no` and `status`; `syslog_messages` now includes `log_time`, `protocol`, `parse_status`, and `retention_expire_at`.
- `employee_devices` now uses `mac_address`/`device_label`; `client_events` now uses `station_mac`, `ap_mac`, `ssid`, IP/hostname fields, `matched_employee_id`, and `match_status`.
- `attendance_records` now includes the richer state fields required by the approved design: `clock_in_status`, `clock_out_status`, `exception_status`, `source_mode`, `version`, `last_calculated_at`.
- `attendance_reports` is now modeled as report history tied to `attendance_record_id` with unique `idempotency_key`, payload/target/response fields, and retry tracking.
- `system_settings` now includes an `id` primary key and seeds the five required default settings rows.
- No remaining mismatches were found in the owned files against the previously identified Task 2 design-alignment issues.

## Task 2 Production Readiness Review
- `git diff --stat 186d066dcbc0f091f1324c540ccd087bc6d10371..1d1e5f518746c62c9cea599de31da601ea4ba183` shows exactly the 7 planned backend files and no scope creep.
- `go test ./...` in `backend/` passes; the new packages compile and the config default test passes.
- The migration matches the design document's recommended fields for `employees`, `employee_devices`, `syslog_messages`, `client_events`, `attendance_records`, `attendance_reports`, and `system_settings`, including the required uniqueness constraints and seed rows.
- File sizes stay small and responsibilities are still narrow: config defaults, bootstrap wiring, migration DDL, and plain domain structs remain separated.
- Potential follow-up concern only: `attendance_reports.payload_json` and `attendance_reports.response_body` are nullable in SQL, while the corresponding domain fields are plain `string`; this is harmless at the current skeleton stage because no persistence code exists yet, but future scanning code must handle nullability explicitly.

## Task 3 Production Readiness Review
- `git diff --stat 1d1e5f518746c62c9cea599de31da601ea4ba183..312f679440296890e7c77fff6fcf2456fc1ad24c` shows exactly the 2 planned parser files and no scope creep.
- `cd backend && go test ./internal/parser -run TestParseConnectEvent -v` and `cd backend && go test ./...` both pass in the current tree.
- Parser and test files remain small and single-purpose: one parses AP syslog into `domain.ClientEvent`, one verifies the connect happy path.
- `ParseAPSyslog` currently ignores the `receivedAt` argument and returns zero values for `domain.ClientEvent.EventDate` and `EventTime`, even though the domain shape requires both and the attendance design aggregates by `attendance_date` and event time.
- `ParseAPSyslog` treats every station log that is not `" connect "` as `"disconnect"`, so unsupported AP messages containing `Station[...]` would be silently misclassified instead of rejected as parse failures.
- Test coverage is too narrow to lock the intended contract: there is no test for `disconnect`, missing station MAC, or unsupported message verbs, so the current misclassification behavior is unguarded.
- The TDD red-to-green sequence is not verifiable from the final commit range alone; only the final passing state is observable.

## Task 3 Re-Review Notes
- Commit `855caa7` still stays within the planned scope and modifies only the two parser files.
- `ParseAPSyslog` now populates `EventDate` with the midnight date derived from `receivedAt` and `EventTime` with `receivedAt` itself.
- Unsupported station messages no longer default to `disconnect`; the parser now returns `unsupported event verb` unless the raw message contains either `" connect "` or `" disconnect "`.
- Tests now cover the four requested scenarios: connect with time assertions, disconnect, missing station, and unsupported verb.
- `cd backend && go test ./internal/parser -v` and `cd backend && go test ./...` both pass in the current tree.
- Implementation remains intentionally narrow: it still only extracts station MAC plus event type/time fields, without adding registry logic, extra abstractions, or non-stdlib dependencies.

## Task 4 Review Scope
- Target commit: `02cdcd4bf4c43ad1b92294cce3002561d1312701`
- Files under inspection:
  - `backend/internal/service/attendance_processor.go`
  - `backend/internal/service/report_service.go`
  - `backend/internal/repository/attendance_repository.go`
  - `backend/internal/repository/report_repository.go`
  - `backend/internal/service/attendance_processor_test.go`
  - `backend/internal/service/report_service_test.go`
- Review focus:
  - 首次 `connect` 是否设置 `FirstConnectAt` 且返回本地 reporting 标志
  - 后续 `connect` 是否保持更早的首次连接时间
  - `disconnect` 是否只保留最新的 `LastDisconnectAt`
  - 幂等键是否由考勤标识、report type、相关时间、version 稳定构造
  - report 创建是否生成 pending 状态与 JSON payload
  - repository 是否仅为最小接口骨架，无 DB/GORM/HTTP/调度器扩展

## Task 4 Review Notes
- Commit `02cdcd4bf4c43ad1b92294cce3002561d1312701` 只新增了规格要求的 6 个文件，没有额外范围扩张。
- `attendance_processor.go` 通过服务本地 `AttendanceProcessResult` 返回 `ClockInNeedsReport`，保持 domain 只读；首次 `connect` 会设置 `FirstConnectAt` 并置报告标志，后续更晚的 `connect` 不会覆盖更早的首次连接，`disconnect` 只会在收到更晚断开时间时更新 `LastDisconnectAt`。
- `report_service.go` 使用 attendance identity、report type、UTC 标准化后的 relevant timestamp 和 `record.Version` 构造稳定幂等键；`CreatePendingReport` 会生成 `domain.AttendanceReport`，并填充 `pending` 状态、幂等键、目标地址与合法 JSON payload。
- `attendance_repository.go` 与 `report_repository.go` 都保持在最小接口骨架级别，只声明查询/保存契约，没有 DB/GORM/HTTP 细节。
- `attendance_processor_test.go` 覆盖了首次 connect 设置打卡与报告标志、后续 connect 不覆盖更早首次连接、disconnect 更新最新断开时间。
- `report_service_test.go` 覆盖了幂等键稳定格式与 pending report 的 JSON payload/状态字段。
- `cd backend && go test ./internal/service -v` 与 `cd backend && go test ./...` 在当前树均通过。
- `AttendanceProcessor.ApplyEvent` 当前会在任意非零 `event.EventTime` 下把 `LastCalculatedAt` 直接写成事件发生时间；对于迟到的旧事件，这个值会被回退到过去，而且该字段语义更像“最近一次计算时间”而不是“当前事件时间”。这是未在 Task 4 需求中声明的附加行为，也没有测试锁定。
- `ReportService` 的 `attendanceIdentity` 在 `record.ID != 0` 时使用 `record-%d`，否则退回 `employee-%d-%s`。这让同一条考勤记录在“未持久化”和“已持久化”两个阶段生成不同的幂等键，削弱了“stable idempotency key”的契约；当前测试只覆盖了有 `record.ID` 的路径，没有覆盖这个边界。

## Task 4 Re-Review Notes
- Re-review target: `97a6cfb0fcd85879f40b385d0617bf0c52e1c6d2` with base `855caa7`.
- `git diff --name-only 855caa7..97a6cfb0fcd85879f40b385d0617bf0c52e1c6d2` 仍然只包含 Task 4 计划内的 6 个文件，没有范围扩张。
- `AttendanceProcessor.ApplyEvent` 已删除对 `LastCalculatedAt` 的写入；当前实现只处理 `EmployeeID`、`FirstConnectAt`、`LastDisconnectAt` 和服务本地 `ClockInNeedsReport`，歧义状态不再由本任务引入。
- `attendance_processor_test.go` 新增断言，锁定首次 connect 时 `LastCalculatedAt` 仍为 `nil`，并新增 `TestApplyEventDoesNotChangeLastCalculatedAt` 验证已有值不会被改写。
- `ReportService.attendanceIdentity` 现在统一基于 `employee_id + attendance_date` 构造，不再依赖 `record.ID`，因此同一业务记录在有无持久化主键时会生成相同幂等键。
- `report_service_test.go` 新增 `TestBuildIdempotencyKeyUsesStableBusinessIdentity`，直接验证同一条业务记录在 `ID=0` 与 `ID=88` 时幂等键保持一致，已锁定此前缺失的稳定性契约。
- `cd backend && go test ./internal/service -v` 与 `cd backend && go test ./...` 在当前树均通过。
- 当前实现仍保持最小职责：repository 仅为接口骨架，service 仅包含状态更新与 pending report 构造，没有引入 DB、HTTP、scheduler 或额外抽象。

## Task 5 Review Scope
- Target commit: `85f5392bae1b9560d6aaebc0e5e6ec56992b5d69`
- Files under inspection:
  - `backend/internal/service/day_end_service.go`
  - `backend/internal/service/day_end_service_test.go`
  - `backend/internal/scheduler/cron.go`
  - `backend/internal/ingest/udp_listener.go`
  - `backend/cmd/server/main.go`
- Supporting files:
  - `backend/internal/bootstrap/app.go`
  - `backend/internal/config/config.go`
  - `backend/internal/domain/attendance.go`
  - `docs/superpowers/plans/2026-03-21-syslog-attendance-implementation-plan.md`

## Task 5 Review Notes
- `git show --stat 85f5392bae1b9560d6aaebc0e5e6ec56992b5d69` 显示提交范围与规格一致，只涉及 5 个计划内文件。
- `DayEndService.FinalizeForDay` 在 `LastDisconnectAt == nil` 时会把 `ClockOutStatus` 设为 `missing`、`ExceptionStatus` 设为 `missing_disconnect`；在已有断开时间时会把 `ClockOutStatus` 从空/pending/missing 归一到 `ready`，因此两个核心状态分支都已实现。
- `UDPListener` 提供了 `Start` 与 `ReadOnce` 两个最小入口，并附带 `Serve`/`Close` 便于后续接线；当前没有引入持久化、HTTP 或第三方依赖。
- `scheduler.Cron` 只是一个 thin wrapper，通过 `RunDayEnd` 批量调用 `DayEndService.FinalizeForDay`，符合“后续可接调度”的最小骨架要求。
- `cd backend && go test ./internal/service -run TestFinalize -v` 与 `cd backend && go test ./...` 在当前树均通过。
- `backend/cmd/server/main.go` 没有实例化现有 `bootstrap.New(...)`，也没有复用 `config.LoadConfigFromEnv(...)` 产出的配置；它改为使用本地 `envOrDefault` 直接读取 `UDP_LISTEN_ADDR`，因此绕过了当前 bootstrap/config 接线要求。
- `backend/internal/service/day_end_service.go` 重新把 `LastCalculatedAt` 写成 `now`，而 Task 5 规格与实现计划中的最小日终逻辑只要求调整 `ClockOutStatus`/`ExceptionStatus`；对应测试也把这个额外行为锁死，属于超出最小范围的新增语义。
- 直接读取当前文件确认上述两个问题的具体位置：
  - `backend/internal/service/day_end_service.go:30` 无条件写入 `LastCalculatedAt`
  - `backend/internal/service/day_end_service_test.go:23-28` 与 `:46-50` 将该额外行为固化进测试
  - `backend/cmd/server/main.go:20-22` 直接读取环境变量并手工构造服务，未使用 `backend/internal/bootstrap/app.go:13-20` 的现有入口

## Task 5 Re-Review Notes
- Re-review target: `299996714d84d2866e3c49083d4b795db4acc63c` with base `85f5392bae1b9560d6aaebc0e5e6ec56992b5d69`.
- `git diff --stat 85f5392bae1b9560d6aaebc0e5e6ec56992b5d69..299996714d84d2866e3c49083d4b795db4acc63c` shows only the three files needed to fix the prior Task 5 issues: `main.go`, `day_end_service.go`, and `day_end_service_test.go`.
- `backend/cmd/server/main.go` now wires through `bootstrap.New(os.Getenv)` and logs `app.Config.Timezone` / `app.Config.SyslogRetentionDays`, so the entrypoint references the existing bootstrap/config path instead of bypassing it.
- `backend/internal/service/day_end_service.go` no longer mutates `LastCalculatedAt`; `now` is accepted but intentionally unused, keeping the service focused on the requested clock-out / exception status transitions.
- `backend/internal/service/day_end_service_test.go` no longer asserts any `LastCalculatedAt` behavior; the tests now only lock the two requested state branches: missing disconnect and existing disconnect.
- `backend/internal/scheduler/cron.go` and `backend/internal/ingest/udp_listener.go` remain unchanged and still match the prior minimal skeleton assessment.
- `cd backend && go test ./internal/service -run TestFinalize -v` and `cd backend && go test ./...` both pass in the current tree.
- No additional persistence, HTTP wiring, third-party scheduler dependency, or other Task 5 scope expansion was introduced in the fix commit.

## Task 5 Requested Range Review Notes
- Review target: `97a6cfb0fcd85879f40b385d0617bf0c52e1c6d2..299996714d84d2866e3c49083d4b795db4acc63c`.
- `git diff --stat` confirms the range adds exactly the 5 planned files, with `main.go` as the only modified pre-existing file; no extra persistence, HTTP, or third-party scheduler code appears in the range.
- Current diff content shows `main.go` now references `bootstrap.New(os.Getenv)`, `ingest.NewUDPListener`, `scheduler.NewCron`, and `service.NewDayEndService`, which satisfies the “wire current bootstrap/config plus new skeleton pieces” requirement without overbuilding.
- `backend/internal/service/day_end_service.go` implements only the two required day-end branches and no longer mutates `LastCalculatedAt`; the tests cover both requested scenarios and stay tightly scoped.
- `backend/internal/ingest/udp_listener.go` exposes the required minimal `Start`/`ReadOnce` surface while keeping lifecycle helpers (`Serve`, `Close`, `Addr`) local to ingest concerns; file size remains reasonable for a single-purpose skeleton.
- `backend/internal/scheduler/cron.go` remains a thin wrapper over `DayEndService.FinalizeForDay`, preserving single responsibility and avoiding a real cron dependency.
- `cd backend && go test ./...` passes in the current tree, so the requested range is buildable and test-clean.
- No blocking production-readiness issue was found in this range. Residual gap only: there are still no direct tests for `udp_listener` or `main.go` wiring, so future behavior changes in startup/listener lifecycle would currently rely on package-level compile coverage rather than explicit tests.

## Task 6 Review Notes
- Review target: `f87d5b2914c6f1e7048395b81535de989549b14f`.
- `git show --stat --name-only f87d5b2914c6f1e7048395b81535de989549b14f` shows exactly the 6 requested files and no extra scope.
- `backend/internal/http/router.go` uses stdlib `http.NewServeMux` and registers the four required GET routes: `/api/attendance`, `/api/employees`, `/api/logs`, `/api/settings`.
- `backend/internal/http/handlers/attendance_handler.go` returns `200 OK` with JSON body `{"items":[]}` via a tiny shared `listResponse`/`writeJSON` helper; no DB, auth, or business logic is introduced.
- `employees_handler.go`, `logs_handler.go`, and `settings_handler.go` each stay thin and return the same placeholder `items: []` payload.
- `backend/internal/http/handlers/attendance_handler_test.go` verifies `/api/attendance` returns `200` with an empty `items` array and also checks that all four admin routes respond with `200`.
- `cd backend && go test ./internal/http/handlers -v` passes in the current tree.
- No spec deviations or scope creep were found in the inspected Task 6 implementation.

## Task 6 Requested Range Production Review Notes
- Review target: `299996714d84d2866e3c49083d4b795db4acc63c..f87d5b2914c6f1e7048395b81535de989549b14f`.
- `git diff --stat` confirms the range adds exactly the 6 planned files, with no unrelated modifications or scope creep.
- `backend/internal/http/router.go` correctly builds a thin stdlib `ServeMux` and registers the four required routes.
- `backend/internal/http/handlers/*.go` remain deliberately small; no handler file is oversized, and the package still contains only placeholder JSON behavior.
- `cd backend && go test ./...` passes in the current tree, so the added package compiles and the handler tests pass.
- However, a repository-wide search for `NewRouter`, `ListenAndServe`, and `net/http` shows the new router is not referenced anywhere outside its own tests. `backend/cmd/server/main.go` still only starts the UDP listener and never creates or runs an HTTP server, so the advertised admin API surface is not actually exposed by the running application.
- The current tests only exercise the router in-memory; they do not catch the missing runtime wiring because nothing asserts that the process starts an HTTP listener or mounts `httpapi.NewRouter(...)`.

## Task 6 Re-Review Notes (Head `19eb2ca287cb8ac0942925750be9681eba050053`)
- Review target: `299996714d84d2866e3c49083d4b795db4acc63c..19eb2ca287cb8ac0942925750be9681eba050053`.
- `git diff --stat` now shows exactly the 7 expected files: the previous 6 HTTP files plus `backend/cmd/server/main.go`; no extra scope was added.
- `backend/cmd/server/main.go` now creates `adminServer := httpapi.NewServer(adminHTTPAddr, httpapi.Dependencies{})`, starts it in a goroutine with `ListenAndServe`, and performs graceful `Shutdown` on exit. This means the running backend process now does expose the admin API through a real stdlib HTTP server.
- `backend/internal/http/router.go` adds `NewServer(addr, deps)` as a thin constructor that sets `Handler: NewRouter(deps)`. `main.go` calls this constructor directly, so runtime wiring goes through the current admin router entry point rather than bypassing it.
- Runtime verification succeeded: starting `go run ./cmd/server` and requesting `http://127.0.0.1:8080/api/attendance` returned `{"items":[]}`. The captured startup log also shows `admin_http=:8080`.
- `backend/internal/http/handlers/attendance_handler_test.go` adds `TestNewServerUsesAdminRouter`, which proves `httpapi.NewServer(...).Handler` serves `/api/attendance` with `200 OK`. This meaningfully locks the `NewServer -> NewRouter` contract, though it still does not directly integration-test `main.go`.
- The implementation remains within Task 6 minimal scope: handlers are still placeholder JSON only, there is no DB/auth/business wiring, and the new HTTP server is stdlib `net/http` without added framework or config expansion.

## Task 7 Review Scope
- Review target: `19eb2ca287cb8ac0942925750be9681eba050053..178427b0b19873d62329116271f45f2e5c7b78ac`.
- Review focus:
  - React + Vite + TypeScript 前端壳是否可运行
  - Shell 是否提供 Dashboard、Logs、Employees、Attendance、Settings 五个导航项
  - Router 是否映射五个页面路由
  - 每个页面是否有明确标题与合适的占位布局
  - 视觉是否具有工业运维控制台气质，而不是普通链接页
  - 测试是否锁定导航渲染且能体现 red -> green 目标
  - 文件职责与体量是否仍保持清晰

## Task 7 Early Notes
- `git diff --stat 19eb2ca287cb8ac0942925750be9681eba050053..178427b0b19873d62329116271f45f2e5c7b78ac` 显示本次变更集中在前端，共 14 个文件、902 行新增、5 行删除，没有扩散到后端。
- 新增结构与计划表面一致：入口、router、shell、5 个页面、全局样式、1 个路由测试，以及 Vite/TS/package 调整。
- 体量上最需要复查的是 `frontend/src/styles.css`（368 行）与 `frontend/src/app/layout/AppShell.tsx`（126 行），需要确认是否仍保持单一职责而不是把展示、路由、设计 token 和页面结构混在一起。
- 当前工作树中的 `frontend/` 已存在 `dist/`、`node_modules/` 和 `tsconfig.tsbuildinfo`；这些不是本次 diff 内容，但会影响“是否可运行”的验证方式，因此需要用命令直接复测 `build` / `test`，不能只凭现成产物判断。
- `frontend/package.json` 引入了完整 React/Vite 依赖集，但仓库跟踪文件列表中没有 `package-lock.json`、`pnpm-lock.yaml` 或 `yarn.lock`，这意味着依赖解析结果不受版本快照约束。
- `frontend/src/test/router.test.tsx` 只渲染了脱离 router 上下文的 `<AppShell />`，实际并未挂载 `router`、未校验 5 个路由页面，也未验证导航点击切换是否工作。

## Task 7 Review Notes
- `frontend/src/app/router.tsx` 正确声明了 5 个目标路由：`/`、`/logs`、`/employees`、`/attendance`、`/settings`，并统一挂载在 `AppShell` 下。
- `frontend/src/features/*Page.tsx` 五个页面都具备明确 `h2` 标题与至少两块占位面板，满足“不是纯文字链接页”的最小骨架要求。
- `frontend/src/styles.css` 的配色、字体系、网格背景和 panel/card 结构基本符合“dark graphite + amber + cold green”的工业控制台方向；源码层面看，设计方向与任务描述一致。
- 运行验证通过：`cd frontend && npm test` 与 `cd frontend && npm run build` 当前均成功。
- 浏览器 MCP 无法启动，因为本机缺少配置要求的 Chrome Beta，可视化抽查未完成；因此视觉结论主要基于 CSS/source review 与构建结果，而不是实际截图验收。
- `frontend/src/test/router.test.tsx` 只锁定 “Dashboard” 文本存在；如果有人删掉 `router.tsx` 中的某个子路由、误改 `NavLink` 目标、或让某个页面不再渲染标题，这个测试仍会继续通过。
- `frontend/src/app/layout/AppShell.tsx` 的 `useInRouterContext()` 条件分支是为了让脱离 router 的测试也能渲染导航卡片，但这引入了运行时并不会走到的备用渲染路径，弱化了组件职责边界和测试真实性。
- `frontend/src/styles.css` 通过 Google Fonts `@import` 拉取字体，前端 shell 的核心视觉会依赖外网字体服务；对内网/受限网络的运维控制台场景，这个依赖存在可用性与合规风险。

## Task 7 Re-Review Scope (Head `150a0b48a29758fb0c36abcc1d75203d5727a234`)
- Review target: `19eb2ca287cb8ac0942925750be9681eba050053..150a0b48a29758fb0c36abcc1d75203d5727a234`.
- Required focus:
  - 测试是否挂载真实 router，并验证 5 个导航项与至少 1 个非默认路由
  - `AppShell` 是否移除 `useInRouterContext()` fallback 分支
  - 样式是否不再依赖 Google Fonts 运行时导入
  - `package-lock.json` 是否已提交并与当前依赖清单匹配
  - 改动是否仍保持在 Task 7 前端壳范围内

## Task 7 Re-Review Early Notes
- `git diff --stat 19eb2ca287cb8ac0942925750be9681eba050053..150a0b48a29758fb0c36abcc1d75203d5727a234` 显示范围仍仅限 `frontend/`，新增文件从 14 个变为 15 个，唯一新增的计划外文件类型是 `frontend/package-lock.json`。
- `git diff --name-only ... -- frontend` 确认没有扩散到 backend/docs/infra，范围控制目前看仍符合 Task 7。
- `session-catchup.py` 用技能安装目录中的脚本执行成功；此前按默认插件路径调用失败是本地技能路径差异，不是项目问题。
- 代码级复核已确认：
  - `frontend/src/test/router.test.tsx` 现在使用 `createMemoryRouter(appRoutes)` + `<RouterProvider />` 挂载真实路由树。
  - 第一条测试显式校验 5 个导航 link 与 `Dashboard` 标题。
  - 第二条测试用 `initialEntries: ["/logs"]` 校验了一个非默认路由，并验证 `Logs` 导航项带有 `aria-current="page"`。
  - `frontend/src/app/layout/AppShell.tsx` 已移除 `useInRouterContext()` fallback 分支，只保留真实 `NavLink` 渲染路径。
  - `frontend/src/styles.css` 已移除 Google Fonts `@import`，源码搜索也未再发现 `fonts.googleapis` 或 `@import url(...)`。
  - `frontend/package-lock.json` 已被 git 跟踪，顶层依赖与 `package.json` 一致。
- 验证过程里曾把 `npm ci` 与 `npm test`/`npm run build` 并行执行，导致后两者在依赖重装中途报错；这属于审查步骤冲突，不是提交本身的问题，需在 `npm ci` 完成后重新验证。

## Task 7 Re-Review Notes
- `frontend/src/app/router.tsx` 现在导出了 `appRoutes`，使测试能够复用与生产一致的路由定义；生产入口仍使用 `router = createBrowserRouter(appRoutes)`，没有引入第二套路由配置。
- `frontend/src/test/router.test.tsx` 满足本次复审的两个核心要求：
  - 首页测试通过真实 router 挂载，并逐个断言 5 个导航项存在。
  - 非默认路由测试使用 `"/logs"`，验证 `Logs` 页面标题与当前导航态。
- `frontend/src/app/layout/AppShell.tsx` 已无 `useInRouterContext()` 或备用 `div` 导航分支；组件职责比上次更干净，测试也不再依赖假路径。
- `frontend/src/styles.css` 已改为系统字体栈；对 `frontend/src` 的源码搜索没有再发现 `fonts.googleapis`、`@import url(...)` 或 `useInRouterContext`。
- `frontend/package-lock.json` 已被 git 跟踪，顶层依赖与 `package.json` 一致；`cd frontend && npm ci` 在当前树成功，说明 lockfile 可实际用于可重复安装。
- 在 `npm ci` 完成后的干净依赖树上，`cd frontend && npm test` 与 `cd frontend && npm run build` 均通过。
- `git diff --stat ... -- frontend` 与 `git diff --name-only ... -- frontend` 都表明本次修改仍局限于 Task 7 前端壳范围，没有扩散到 backend 或其他任务区域。
- 本次复审未发现新的阻塞问题。前次指出的四个问题：真实 router 测试缺失、`AppShell` fallback 分支、Google Fonts 运行时依赖、缺少 lockfile，均已关闭。
