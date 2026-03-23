# Findings & Decisions

## Feishu Notification Timezone Fix (2026-03-23)
- 飞书通知文案里的“日期/时间”此前使用 `time.Local` 渲染，而不是应用配置的固定时区；当 Ubuntu 服务器系统时区为 UTC 时，消息文本会显示 UTC 时间。
- 这个问题只影响通知文案展示，不影响飞书打卡导入本身的 `check_time` 秒级时间戳。
- 最小正确修复是给 `AttendanceReportDispatcher` 显式注入应用 `Location`，并在 `buildAttendanceNotificationText` 中统一使用该时区渲染文本。
- bootstrap 已将 `app.Location` 传入 dispatcher，避免继续依赖主机 OS 时区，符合 KISS 和单一职责。

## Syslog Rule Sort Order Migration Fix (2026-03-22)
- 启动报错 `Unknown column 'sort_order' in 'field list'` 的根因不是 repository 提前读表，而是 migration 执行顺序错误。
- `001_init.sql` 在老库场景下会先执行 `INSERT IGNORE INTO syslog_receive_rules (sort_order, ...)`，而 `sort_order` 的幂等补列 SQL 写在这条 insert 后面；旧表没有该列时，migration 会在 seed insert 处提前失败，根本到不了后续 `ALTER TABLE`。
- 修复方式保持最小化：
  - 将 `syslog_receive_rules.sort_order` 的补列与回填块前移到 seed insert 之前
  - 新增回归测试锁定顺序：`TestMigrationAddsSyslogRuleSortOrderBeforeSeedInsert`
- 这样既兼容新库首次建表，也兼容已有库增量补列，符合 KISS，避免为单文件 migration 额外引入复杂迁移框架。

## Feishu Interaction Logging Findings (2026-03-21)
- 当前飞书链路真正缺的不是功能，而是可观测性：之前 `tenant_access_token`、`batch_create`、`batch_del` 的请求/响应不会进入后端日志，排查时只能回看 `attendance_reports.response_body`。
- 本轮日志增强采用两层结构：
  - `AttendanceReportDispatcher` 记录业务阶段日志：开始 dispatch、删旧开始、create 开始、成功、失败
  - `FeishuAttendanceHTTPClient` 记录接口层日志：HTTP request/response、token cache hit、token 获取成功/失败、create/delete 成功/失败
- 日志安全边界必须前置，而不是“先打印再补救”：
  - `Authorization` 统一打印为 `Bearer ***`
  - `app_secret` 统一打印为 `***`
  - `tenant_access_token` 统一打印为 `***`
- 为了把脱敏逻辑锁住，本轮新增了 `feishu_attendance_client_test.go`，专门验证 header/body/response 的脱敏 helper，不依赖手工肉眼检查日志。
- 这次没有引入新的日志库，继续复用标准库 `log.Printf`，符合当前项目 KISS 原则；后续如果全局迁移到结构化 logger，再把这些日志迁过去即可。

## Frontend Debug Tools Findings (2026-03-21)
- 调试能力最稳的落点不是把逻辑塞回现有 `LogsPage` / `AttendancePage`，而是新增独立 `调试工具` 页。这样符合 SRP，也避免把正式业务页继续变成“半业务半运维”的混合界面。
- 手工 syslog 注入如果只接收原始报文而不允许指定 `receivedAt`，就无法复现历史、跨天和补录场景。当前 parser 以 `receivedAt` 作为 `EventDate/EventTime` 的唯一来源，因此调试入口必须允许手填接收日期时间。
- 浏览器无法直接发 UDP，也不该由前端直接调用飞书接口；最小正确方案是新增后端 debug API：
  - `POST /api/debug/syslog`
  - `POST /api/debug/attendance/{id}/dispatch`
- 手工飞书发送如果直接调用后台 `RunOnce()`，会顺带处理整批 pending/failed report，副作用过大。最终实现通过给 `AttendanceReportDispatcher` 增加单条 `DispatchReport(...)` 入口，保证“点哪条就只发哪条”。
- 手工飞书发送不能复用自动上报的 canonical `idempotency_key`，否则重复点击可能只是在覆盖旧本地 report 而不产生真正的重发。最终实现新增 `CreateManualPendingReport(...)`，在 canonical key 后追加 `/manual/<requestedAt>`，允许显式重发。
- 为了保持飞书远端状态与当前考勤记录一致，手工发送仍复用“删旧再导新”语义：会查找同一 `attendance_record_id + report_type` 最近一次成功的 `external_record_id`，回填到 `delete_record_id` 后再 dispatch。
- 手工 syslog 注入的解析反馈不需要新建第二套 parser 结果模型；调试 service 只预解析一次拿 `parseStatus/parseError`，真正落库仍统一走 `SyslogPipeline.Handle(...)`。
- 调试 UI 按已确认决策保持始终可见，但所有动作都做二次确认。为践行 KISS，本轮使用原生 `window.confirm`，不引入新的 modal 组件和状态机。
- 本轮验证结果：
  - `cd backend && go test ./...` 通过
  - `cd frontend && npm test -- --run` 通过（10 files, 21 tests）
  - `cd frontend && npm run build` 通过
- 本轮未新增数据库迁移，也未修改既有业务 API 契约；新增行为全部收口在新的 debug API、前端独立调试页和 dispatcher 的单条 dispatch 抽象中。

## Feishu Attendance Reporting Findings (2026-03-21)
- 现有系统已经有 `attendance_reports` 队列表和 `ReportService`，但只有“生成 pending report”的逻辑，没有真正派发到外部系统的 worker；这是接飞书的最小可复用基础。
- 当前 `SyslogPipeline` 与 `AttendanceAdminService` 仍依赖 `report_target_url` 设置项来决定是否生成 pending report。若直接在业务层拼飞书请求，会让考勤计算逻辑和外部系统耦合过深，不符合 SRP；因此应保留“内部 report 事件 + 独立 dispatcher”边界。
- 官方飞书接口已核实：
  - 自建应用 token：`POST /open-apis/auth/v3/tenant_access_token/internal`
  - 导入打卡流水：`POST /open-apis/attendance/v1/user_flows/batch_create?employee_type=employee_id`
  - 删除打卡流水：`POST /open-apis/attendance/v1/user_flows/batch_del`
- 飞书导入成功响应会返回 `record_id`，删除又必须依赖 `record_ids`；因此若系统要支持人工修正/清除后的“删旧再导新”，必须把飞书返回的 `record_id` 持久化到本地 report。
- 员工主数据当前没有飞书员工标识；若继续复用 `employee_no` 或 `system_no` 会把内部编号和外部人事编号耦死，扩展性差。新增独立 `feishu_employee_id` 字段更符合 DIP 和后续系统演进。
- 系统设置的 admin service 会先读取 `system_settings` 现有键，再拒绝未知 key，因此只要 migration 插入新的飞书设置键，现有 handler/更新流程就能复用，无需新建第二套设置接口。
- 现有 correction 流程已经把“真实变化才生成 report”收口好了，但还没有“找到上一次成功外部流水并保存为 delete_record_id”的能力；这部分应该下沉到 report/service 层，不应散落在 handler。
- 最终实现决策：
  - 员工新增 `feishu_employee_id`，数据库采用 `NULL + UNIQUE`，业务层仍要求新建/编辑必须填写，从而兼容历史空值并避免唯一索引与空字符串冲突。
  - `attendance_reports` 保留原有 `target_url` 列做兼容，但新逻辑不再读取 `report_target_url`；真正的飞书同步改由 `external_record_id / delete_record_id` 跟踪外部流水。
  - `AttendanceAdminService` 在生成人工修正或 clear report 时，会查询同一 `attendance_record_id + report_type` 的最近一次成功飞书流水，并把其 `external_record_id` 回填到新 report 的 `delete_record_id`。
  - 新增 `AttendanceReportDispatcher` 后台 worker，按 `pending/failed + retry_count < report_retry_limit` 拉取 report；clear 只删不导，其余 report 先删旧再导新。
  - 新增 `FeishuAttendanceHTTPClient`，负责 `tenant_access_token` 获取与缓存、`batch_create` / `batch_del` 调用，并把飞书响应 `record_id` 回写到 `attendance_reports.external_record_id`。
  - 前端员工档案和系统设置已经切到飞书字段：`feishuEmployeeId`、`feishu_app_id`、`feishu_app_secret`、`feishu_creator_employee_id`、`feishu_location_name`；旧 `report_target_url` 已从 UI 中移除。
- 本轮验证结果：
  - `cd backend && go test ./...` 通过
  - `cd frontend && npm test -- --run` 通过（9 files, 18 tests）
  - `cd frontend && npm run build` 通过
- 当前残余风险：
  - `001_init.sql` 仍是单文件反复执行模式，本轮通过 `INFORMATION_SCHEMA + PREPARE` 做了幂等补列；后续如果 migration 继续复杂化，最好升级成带版本表的正式迁移机制。
  - `tenant_access_token` 只做进程内缓存，适用于当前单实例部署；如果后续扩成多实例，需要把 token 缓存或派发幂等策略外移。

## Frontend Command Deck Redesign Findings (2026-03-21)
- 当前工作区存在大量未提交改动，且前端相关文件已处于修改状态；本轮需在现有工作树上增量实现，避免覆盖后端和既有前端调整。
- 现有前端核心问题不是功能缺失，而是样式分散：
  - `frontend/src/styles.css` 只提供了一层粗粒度深色视觉
  - `EmployeesPage`、`AttendanceTable`、`SettingsForm` 等关键交互大量依赖内联样式对象
  - 页面之间缺少统一的组件语义、密度规则和状态表达
- 已确认本轮设计决策：
  - 信息架构：`Command Deck`
  - 主题：`Cold Slate`
  - 模式：白天/夜间双主题
  - 终端：桌面优先
  - 密度：平衡密度
  - 文案：全部中文
- 现有测试不是简单快照，而是带业务语义的 RTL 测试；本轮重构必须同步维护：
  - 路由导航名称与激活态
  - Dashboard 的汇总推导与异常展示
  - Logs 的搜索、分页、轮询、新消息提示
  - Attendance 的修正行为与可编辑边界
  - Employees 的创建/编辑/停用 API 调用
  - Settings 的加载、禁用、校验、保存行为
- 后端 API 契约保持不变；新增行为只允许落在前端主题状态与展示层，不引入新的后端字段。
- 第一轮 TDD 已完成并验证：
  - `router.test.tsx` 已改为新的中文导航与页面标题约束
  - 新增 `app-shell.test.tsx` 锁定 `system/light/dark` 三态主题切换与 `localStorage` 持久化
  - 当前实现已让 `AppShell` 承担主题状态管理，并通过 `document.documentElement.dataset.theme` 驱动样式
- 第一层实现决策已落地：
  - 壳层采用左侧导航 + 顶部指挥栏 + 摘要信号条
  - 主题状态不引入全局状态库，先由 `AppShell` 本地管理即可满足当前路由结构
  - 全局样式已切到 `Cold Slate` 令牌体系，并开始替换旧的英文导航与标题
- 最终实现结果：
  - `AppShell` 已切换到 `Command Deck` 三段式壳层，新增 `system/light/dark` 三态主题切换，并通过 `localStorage + data-theme` 持久化
  - `Dashboard / Logs / Attendance / Employees / Settings` 五页已完成中文化、双主题化与统一面板/表格/表单语义
  - `EmployeesPage` 从“列表内嵌编辑”改为固定编辑工作区，`AttendanceTable` 与 `SettingsForm` 已去掉核心内联样式，复用共享 class 语义
  - 样式层已补齐响应式压缩规则、按钮/输入焦点态与小屏降级布局，满足“桌面优先，移动端可访问”的目标
- 本轮验证结果：
  - `cd frontend && npm test -- --run` 通过（9 files, 18 tests）
  - `cd frontend && npm run build` 通过
- 当前残余风险：
  - 初版改造时字体通过 Google Fonts `@import` 加载；该问题已在后续调整中移除，现改为本机字体栈，不再依赖外网字体服务
- 后续视觉调整结果（2026-03-21）：
  - 日志页已从左右双栏改为上下结构：过滤条件在上、日志明细在下，更符合连续检索和浏览行为
  - 字体已切换为本机字体栈：中文正文优先 `PingFang SC / Microsoft YaHei UI`，等宽信息使用本机 monospace 栈
  - 本轮未改动日志 API、分页、轮询或新消息提示逻辑，仅调整布局与字体来源
- 日志页细化结果（2026-03-21）：
  - `接收时间` 已升级为完整 `接收日期时间` 展示，直接显示完整日期与时分秒
  - `站点 MAC` 列已调宽，并补充内容换行规则，避免文本溢出导致底框视觉错位
  - `原始消息` 已从表格直出改为 `详情` 按钮，点击后通过标准 dialog 弹层查看完整日志内容
  - 本轮仍未改动日志接口与分页轮询，仅收敛明细呈现方式和信息密度
- 日志页继续收敛结果（2026-03-21）：
  - 过滤条件已新增 `开始日期 / 结束日期`，并接到真实 `logs` 查询参数链路，而不是前端假过滤
  - 后端 `GET /api/logs` 已支持 `fromDate / toDate`，仓储层通过 `received_at` 日期范围做过滤
  - 日志明细表已去掉 `主机` 列，新增 `员工` 列；员工名字由前端基于 `station MAC -> employee.devices` 本地映射得到
  - 接收日期时间已保持单行展示，表格密度更稳定
  - 详情弹层同步展示员工名字，不再依赖主机字段

## Task 12 Review Scope (2026-03-21)
- Review baseline: `ad1e707eb488b8961c6f6e4951a985853f23d37b`
- Review target: current `HEAD`
- Stated primary files:
  - `backend/internal/http/router.go`
  - `backend/internal/http/handlers/attendance_handler.go`
  - `backend/internal/http/handlers/attendance_handler_test.go`
  - `backend/internal/http/handlers/attendance_handler_internal_test.go`
  - `backend/internal/http/handlers/employee_admin_handlers.go`
  - `backend/internal/http/handlers/settings_handler.go`
  - `backend/internal/http/handlers/admin_handlers_test.go`
  - `backend/internal/service/admin_services.go`
  - `backend/internal/service/admin_*_test.go`
  - `backend/internal/repository/employee_repository.go`
  - `backend/internal/repository/attendance_repository.go`
  - `backend/internal/repository/system_setting_repository.go`
  - `backend/internal/repository/mysql_repository_test.go`
  - `backend/internal/bootstrap/app.go`
  - `backend/internal/domain/attendance.go`
- Review focus:
  - employee + devices transaction consistency and update semantics
  - manual correction patch semantics, `version/status`, report generation, `target_url`
  - settings bulk update constraints
  - handler request / response contract and error handling
  - whether tests lock the behavior sufficiently

## Task 12 Review Findings (2026-03-21)
- `AttendanceAdminService.CorrectAttendance` 支持 `null` 清空时间字段，但只会在“提供了非 null 时间”时生成 report；这意味着清空已上报时间只会改本地记录和 `version`，不会产生任何对外补偿动作，外部系统状态可能长期滞后。
- `AttendanceAdminService.CorrectAttendance` 在任何提供字段的请求上都会先 `record.Version++`，然后按提供字段生成新 report；即使请求值与现有值完全相同，也会生成新的版本和新的幂等键，导致无实际修正的重复上报。
- 员工写接口的 repository/service 没有检查 `UPDATE` / `Disable` 的 `RowsAffected`。`PUT /api/employees/{id}` 在目标不存在且 `devices` 为空时会返回成功，`POST /api/employees/{id}/disable` 对不存在员工也会返回 200。
- 员工创建/更新成功后返回的是内存中拼装的 `devices`，没有回读数据库，因此返回体中的 device `id/createdAt/updatedAt` 都是零值，与 `GET /api/employees` 的契约不一致。
- handler 错误映射只区分三类 validation error，`sql.ErrNoRows`、唯一键冲突等都会落成 500，新的 admin 写接口缺少 404/409 等更准确的 HTTP 语义。
- settings 批量更新已限制“只能更新已知 key”，并且放进事务里逐项保存；但没有拒绝同一批次重复 key，当前语义是“后写覆盖前写”。
- 测试覆盖了 happy path、事务回滚和基本路由联通，但没有锁定以下关键行为：attendance `null` patch、attendance no-op patch、employee not-found、employee 写接口返回值完整性、settings 重复 key、handler 错误状态码。

## Task 12 Final Outcomes (2026-03-21)
- 提交 `ac998e28cf1d4f5a2f395aab5521422dbfe27aa5` 修复了 admin 写接口的核心边界：
  - attendance manual correction 只在真实变化时落库并增版本
  - `null` clear 语义会生成 `action="clear"`、`timestamp=null` 的 pending report
  - 员工写接口回读真实持久化状态
  - handler 补齐 `400/404/409/500` 的基本错误语义
  - settings 批量更新拒绝重复 key
- 提交 `c703ecd48da8060f480eb2ad8f464b9b4624e140` 修复了剩余阻塞点：
  - 去掉 repository 层对 `RowsAffected()==0` 的 not-found 误判
  - service 层增加 employee 存在性预检，保证“主字段不变但只换 devices”仍能成功
  - 已 disabled 员工再次 disable 不再错误返回 404
  - attendance correction handler 的 404 路径已补测试
  - duplicate key 409 判定收窄到 MySQL `1062`
- Task 12 已通过二次 spec review 与 quality review，质量评估结论为 `Ready to merge? Yes`。

## Task 12 Fix Review Scope (2026-03-21)
- Review baseline: `1206423b0b5bd4c5314eed6cd56808ea5176700c`
- Review target commit: `ac998e28cf1d4f5a2f395aab5521422dbfe27aa5`
- Files in scope:
  - `backend/internal/http/handlers/admin_handlers_test.go`
  - `backend/internal/http/handlers/response.go`
  - `backend/internal/repository/employee_repository.go`
  - `backend/internal/repository/mysql_repository_test.go`
  - `backend/internal/service/admin_attendance_service_test.go`
  - `backend/internal/service/admin_employee_service_test.go`
  - `backend/internal/service/admin_services.go`
  - `backend/internal/service/admin_settings_service_test.go`
  - `backend/internal/service/report_service.go`
  - `backend/internal/service/report_service_test.go`
- Primary checks:
  - attendance correction no-op / clear-null / idempotency / transaction semantics
  - employee write not-found / persisted response / transaction and errors
  - handler error mapping and trailing JSON rejection
  - tests for key edge cases

## Task 12 Fix Review Findings (2026-03-21)
- 上轮关于 attendance no-op patch、clear/null patch、employee 返回真实持久化状态、settings duplicate key、404/409 映射、trailing JSON 的主要问题，这次提交都做了针对性修复，方向基本正确。
- 新的阻塞问题出现在 employee not-found 修复本身：`employee_repository.Update/Disable` 以 `RowsAffected()==0` 判定 `sql.ErrNoRows`。在 MySQL 默认语义下，命中行但值未变化时也可能返回 0，导致幂等 update、重复 disable，甚至“只改 devices、employee 主字段未变化”的请求被误判为 404 并提前回滚。
- attendance correction 的 no-op 提前返回逻辑已消除“无实际修正仍增版本/落库/发 report”的问题；clear patch 也会生成显式 `action:"clear"`、`timestamp:null` 的补偿 report。
- attendance correction 的 clear patch 仍存在一个残余风险：状态重算只覆盖 `pending/done/missing_disconnect/none` 这组状态，没有为“仅清掉 firstConnectAt、仍保留 lastDisconnectAt”这样的局面定义更明确的异常状态；当前会得到 `clockInStatus=pending`、`clockOutStatus=done`、`exceptionStatus=none`。
- handler 层的 404/409/400 映射比上轮稳了，但测试仍未覆盖 attendance correction 的错误映射路径，也没有覆盖 employee disable 的 404 或 settings 的 409/500 分支。
- 测试新增了 no-op、clear patch、employee not-found、duplicate key、trailing JSON 等场景，但没有锁住“employee 主字段不变但 devices 变化”与“disable 已经是 disabled”这两个最容易被 `RowsAffected` 误判的路径。

## Task 12 Follow-up Review Scope (2026-03-21)
- Review baseline: `ac998e28cf1d4f5a2f395aab5521422dbfe27aa5`
- Review target commit: `c703ecd48da8060f480eb2ad8f464b9b4624e140`
- Files in scope:
  - `backend/internal/http/handlers/admin_handlers_test.go`
  - `backend/internal/http/handlers/response.go`
  - `backend/internal/repository/employee_repository.go`
  - `backend/internal/repository/mysql_repository_test.go`
  - `backend/internal/service/admin_employee_service_test.go`
  - `backend/internal/service/admin_services.go`
- Primary checks:
  - `RowsAffected==0` 误判是否解除
  - employee 主字段不变但 devices 变化的更新路径
  - disable 已 disabled 员工的语义
  - 新存在性预检是否引入新问题
  - attendance correction handler 404 覆盖
  - duplicate key 409 判定稳健性

## Task 12 Follow-up Review Findings (2026-03-21)
- 上轮 Critical 的 `RowsAffected==0` 误判已经解除：repository 恢复为单纯执行 `UPDATE`，not-found 判定前移到 service 层的 `FindByID` 预检。
- employee 主字段不变但 devices 变化的更新路径已被修复，并且有针对性测试覆盖：预检确认存在后，即便 `UPDATE employees` 返回 0 affected，也会继续替换 devices 并提交事务。
- disable 已 disabled 员工不再误报 404：存在性由预检承担，后续 `UPDATE employees SET status='disabled'` 返回 0 affected 也不会被判定为 not-found。
- attendance correction handler 的 404 映射已补上路由级测试，能覆盖 `sql.ErrNoRows -> 404` 的 handler 语义。
- duplicate key 409 判定比上一轮更稳：已移除基于错误文案 `contains("duplicate")` 的宽松兜底，只保留 MySQL 1062 的结构化判定。
- 本轮未发现新的阻塞问题。存在性预检引入了一次事务外查询和一个很小的并发窗口，但在当前系统没有 employee hard-delete 路径的前提下，更像残余设计取舍，不构成这轮阻塞项。

## Task 13 Frontend Review Findings (2026-03-21)
- Critical:
  - `SettingsPage` 在初次 `/api/settings` 返回前就渲染了可提交的表单，并用 `DEFAULT_VALUES` 作为实际提交源。用户如果在加载完成前点击保存，会把占位默认值整包写回后端，覆盖真实系统配置。
- Important:
  - `AttendanceTable` 把 `exceptionStatus === "corrected"` 也判成“异常待处理”，因此修正后的记录仍然显示可编辑输入框和“提交修正”按钮；分支里的“已归档”文案对 corrected 行实际上是死代码。
  - `lib/api.ts` 只对 settings/logs 做了大小写字段归一化，employees/attendance 仍按前端自定义类型直接信任后端；真实后端返回的是数值型 `id`/`employeeId`，但测试全部用了字符串，导致“真实契约已对齐”的结论并不稳。
  - 新测试没有覆盖 `Settings` 最危险的时序路径：初次加载未完成就提交；也没有用真实后端风格的数值型 employee/attendance ID 去验证页面 join 与更新流程。
- Minor:
  - `fetchMock.assertAllMatched()` 没有在任何测试里调用，测试路由表允许残留未消费的 mock 而不失败。
  - `SettingsForm` 对时间只校验 `HH:MM` 形状，不校验合法范围；当前 invalid test 之所以通过，是因为同一次提交里还把保留天数改成了 `0`。

## Task 13 Final Outcomes (2026-03-21)
- 提交 `d2e8919e8af3d323d1d664701613f28dac1d1aea` 将前端从 mock 数据层切到真实后端 HTTP API，覆盖员工、考勤、设置、日志和总览页面。
- 提交 `b0a65c2e19f886bdfa70747d848d75044afd87ae` 修正了前端对后端真实考勤状态语义的误读，避免把正常记录误判为异常。
- 提交 `6003b14` 补齐了前端对 employees / attendance numeric ID 的归一化，并在 settings 首次加载完成前禁用提交，降低默认值误写风险。
- 提交 `6df2937` 收紧了 settings 失败态保护与 attendance attention 语义：首次配置加载失败时仍禁止保存，`AttendanceTable` 与 `Dashboard` 统一复用同一套待处理原因文本，`pending clock-in` 会展示为 `clock_in_pending`，而已恢复正常的 manual 记录不会继续停留在待处理集合。
- Task 13 最终已通过 spec review 与 quality review，quality reviewer 结论为 `Ready to merge? Yes`。

## Frontend Follow-up Review Findings (2026-03-21)
- Critical:
  - `SettingsPage` 的“加载完成前可提交默认值”问题仍未修复。当前 head 依旧会在初次加载阶段渲染可提交表单，并把 `DEFAULT_VALUES` 作为保存载荷来源。
- Important:
  - `AttendanceTable.requiresAttention` 与 `DashboardPage` 的 attention 统计都把 `sourceMode === "manual"` 直接等同于“仍待处理”。这会让已经人工修正且状态正常的记录永久停留在可编辑/待处理集合里，污染考勤列表、Dashboard 指标和健康度。
  - `lib/api.ts` 仍未对 employees / attendance 做真实契约归一化；真实后端的数值型 `id` / `employeeId` 继续被前端当成字符串类型消费。
  - 跟进后的 attendance / dashboard tests 只改了样例状态值，没有补“manual 但已恢复正常”的关键边界，也没有修正 string ID 假设，因此没锁住这轮最容易出错的行为。
- Minor:
  - `fetchMock.assertAllMatched()` 依旧未被任何测试调用。

## Frontend Commit 6003b14 Review Findings (2026-03-21)
- Critical:
  - `SettingsPage` 现在只在“加载中”阶段禁用了表单，但在首次加载失败后会立即解锁，并继续把 `DEFAULT_VALUES` 作为当前可提交值。用户在“设置装载失败，请稍后重试”后点击保存，仍会把占位默认值整包写回后端。
- Important:
  - `requiresAttendanceAttention` 把 `clockInStatus === "pending"` 视为待处理，但 `AttendanceTable` 的异常列和 `Dashboard` 的 watch item 文案仍主要展示 `exceptionStatus`。后端人工修正路径能产生 `clockInStatus="pending"`, `clockOutStatus="done"`, `exceptionStatus="none"` 的真实记录；这类记录会被算进待处理，却在列表和总览文案里显示“无异常 / none”，语义自相矛盾。
  - 新增测试虽然补了 numeric ID、manual resolved、assertAllMatched 和加载禁用，但没有覆盖“settings 首次加载失败后仍可保存默认值”以及“pending clock-in + exception none”这两个关键边界。

## Frontend Final Fix Review Findings (2026-03-21)
- Review baseline: `b0a65c2e19f886bdfa70747d848d75044afd87ae`
- Review target commit: `6003b14dcdadb6be8593cddaddb85f51314bb120`
- 净变更仅涉及 10 个前端文件，集中在 `SettingsPage` / `SettingsForm`、`api.ts`、attendance/dashboard attention 逻辑及对应测试；未发现跨后端、路由或无关页面的范围漂移。
- `SettingsPage` 新增 `isLoading`，并把 `isLoading` 透传给 `SettingsForm.isDisabled`；`SettingsForm` 又将其用于所有输入和 submit 按钮，因此首次加载完成前 UI 不可提交。
- `api.ts` 新增 `stringIdValue`、`normalizeEmployee`、`normalizeAttendanceRecord`，并将 employees / attendance 列表与写接口响应统一映射为字符串 ID；这与后端 domain 的 `uint64` JSON 输出契约相匹配。
- 新增 `requiresAttendanceAttention()` 作为共享判定，`AttendanceTable` 与 `DashboardPage` 都改为复用它；规则已移除 `sourceMode === "manual"` 这个错误条件，因此“manual 但状态已恢复正常”的记录不再进入待处理集合，两个页面语义保持一致。
- 测试已补足本轮关键边界：
  - `settings-page.test.tsx` 覆盖加载完成前按钮禁用；
  - `api.test.ts` 覆盖 numeric employee / attendance ID 归一化；
  - `attendance-page.test.tsx` 与 `dashboard-page.test.tsx` 覆盖 manual-resolved 记录不再显示待处理。
- 验证结果：
  - `cd frontend && npm test -- --run` 通过（7 files, 12 tests）
  - `cd frontend && npm run build` 通过
- 本轮未发现新的 spec 差异或需要阻塞的残余风险。

## Frontend Final Fix Quality Re-Review Findings (2026-03-21)
- `frontend/src/lib/api.ts` 把 mixed numeric/string ID 适配放在 `response.json()` 之后处理，只能覆盖“小整数”场景。后端真实契约里的 employee / device / attendance ID 是 `uint64` JSON number；一旦值超过 JavaScript safe integer，`JSON.parse` 阶段就会丢精度，`stringIdValue()` 只会把错误数字转成错误字符串。新增 `api.test.ts` 只覆盖了 `101/9001` 这种小数值，没有锁住这一边界。
- `SettingsForm` 的 loading/save 状态机仍不完整：所有输入只跟 `isDisabled`（加载态）联动，而不跟 `isSaving` 联动；保存期间按钮会禁用，但输入仍可编辑。`SettingsPage` 在保存成功后又会用返回值更新 `initialValues`，借助 `key={JSON.stringify(initialValues)}` 重挂整个表单，因此用户在请求飞行期间输入的新值会被静默丢弃。现有 settings 测试只覆盖“加载前禁用”“正常保存”“非法输入”“加载失败”，没有锁住 save-in-flight 行为。

## Frontend Commit 6df2937 Review Findings (2026-03-21)
- 本轮聚焦项未发现新的实际问题。
- `SettingsPage` 通过 `hasLoadedSettings` 把“加载中 / 已加载 / 加载失败”三种状态分开，首次加载失败时表单保持禁用，默认值覆盖风险已收住。
- `attendance` 待处理语义已收敛到共享 helper：`AttendanceTable` 与 `Dashboard` 都使用 `getAttendanceAttentionReason` / `requiresAttendanceAttention`，对 manual resolved 和 pending clock-in 两类边界给出一致结果。
- 新增测试已覆盖 initial load fail 与 pending clock-in 两个关键场景，能够锁住这轮修复目标。

## Requirements
- 用户要求执行 `/init`，从零创建一个 syslog 服务端项目。
- 前端技术栈指定为 React。
- 后端技术栈指定为 Golang。
- 数据库指定为 MySQL。
- 核心能力至少包括：接受消息、处理消息、储存消息、发送消息。
- 用户选择交付档位 `C`，希望首版即覆盖较完整能力，而不只是骨架或最小闭环。
- 接收端需要同时支持 `RFC 3164 + RFC 5424`，并具备自动识别能力。
- syslog 输入源是 AP 设备发出的日志，日志中包含客户端连接和断开事件。
- 数据库需要维护员工工号、系统编号、员工姓名、员工设备 MAC 地址（一个员工可绑定多个 MAC）。
- 业务处理模块需要根据 AP 日志分析当日首次连接时间作为上班打卡、当日最后一次断开时间作为下班打卡。
- 考勤分析结果需要通过指定 API 上报到外部服务器。
- 员工名下存在多个设备 MAC 时，考勤按员工维度聚合：取当天所有设备中的最早连接时间作为上班、最晚断开时间作为下班。
- 外部 API 当前没有固定协议，首版可由系统定义一个可配置的 `HTTP JSON` 上报格式。
- 首版接入层只需要支持 `UDP 514`。
- 员工资料与设备 MAC 在首版通过前端界面手工维护。
- 前端首版需要同时覆盖运维监控、资料维护和考勤结果视图。
- 上班时间规则：员工当天第一次连接后立即上报。
- 下班时间规则：需等待到用户配置的日终时间（默认可理解为 23:59）后，取当天最晚断开时间再上报。
- 日终汇总上报时间需要由前端提供配置能力。
- 如果员工当天没有断开事件，到日终时间时不自动上报下班，留待人工处理。
- 系统统一按 `Asia/Shanghai` 时区处理当天归属、上报时间和调度任务。
- AP 日志中可以直接解析到客户端 MAC，字段格式为 `Station[xx:xx:xx:xx:xx:xx]`。
- AP 日志事件类型可直接从消息文本中的 `connect` / `disconnect` 识别。
- 当前已知日志样例包含 AP MAC、客户端 MAC、SSID、IP、hostname 等字段，可作为原始日志和解析结果存储。
- 首版需要支持管理员手工修正某天的上下班时间，并允许修正后重新上报。
- 原始 syslog 不做永久保存，只保留固定天数。
- 外部 API 上报必须具备幂等保护，避免同一考勤结果重复发送。
- 前端首版不做登录鉴权，按内网单管理员工具实现。
- 首版只配置一个外部服务器作为考勤上报目标。

## Research Findings
- 当前项目目录为空，没有现有代码、文档或脚手架可复用。
- 当前目录还不是 git 仓库，说明需要按初始化项目处理。
- 因为包含前端、后端、数据库和协议能力，这属于多阶段复杂任务，需要持续规划和记录。
- 选择 `C` 后，范围已经扩大到多个独立子系统，后续设计需要先做边界收敛，避免一次性过度实现。
- 领域核心已从“syslog 消息中转”转向“AP 在线事件识别 + 员工设备映射 + 考勤上报”。
- 当前 AP 日志结构属于半结构化文本，首版可以用明确字段提取策略实现，后续再扩展多厂商适配层。

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| 优先明确最小可交付范围和协议范围 | syslog 实现边界会直接影响后端结构、数据库模型和前端功能，先定边界更符合 KISS/YAGNI |
| 先确认协议兼容目标，再决定转发与前端字段设计 | RFC 3164 / RFC 5424 的支持程度会影响 parser、存储结构和 UI 展示复杂度 |
| 首版按双协议兼容设计消息模型 | 既符合用户选择的完整首版，也避免后续因字段不足重构存储层 |
| 增加独立的信息处理模块 | 将协议接入与业务判定解耦，符合 SRP，便于后续扩展更多考勤或告警规则 |
| 输出能力定义为外部 API 上报 | 与用户描述一致，避免误建 syslog 转发器 |
| 考勤聚合按员工维度进行，而不是设备维度 | 符合用户业务规则，也能正确处理一个员工绑定多个 MAC 的场景 |
| 首版外部上报采用可配置 `HTTP JSON` 协议 | 先保证业务闭环，降低外部耦合，符合 YAGNI，后续再扩展实际接口适配器 |
| 首版仅实现 UDP syslog 接入 | 满足当前需求的最小必要传输层，避免 TCP/TLS 提前复杂化 |
| 员工与设备资料首版通过后台页面手工维护 | 范围更可控，先保证基础资料管理闭环 |
| 前端首版采用运维与业务并重的控制台形态 | 与用户选择一致，首页需要同时承载状态监控、配置和考勤视图 |
| 上班和下班采用分阶段上报策略 | 满足业务规则，避免把未最终确定的离线时间过早上报 |
| 无断开事件时不自动推断下班时间 | 业务上更保守，避免错误考勤数据污染外部系统 |
| 系统固定按 `Asia/Shanghai` 时区处理日切与任务调度 | 与用户明确要求一致，避免跨时区导致当天归属错误 |
| 首版解析器按当前 AP 日志格式提取字段 | 已有明确样例，直接实现最小必要规则，符合 KISS/YAGNI |
| 首版纳入人工修正能力 | 满足异常考勤处理场景，避免系统只能展示异常却无法闭环 |
| 原始 syslog 采用有限保留策略 | 满足追溯需求的同时控制 MySQL 体量，符合 YAGNI |
| 外部上报必须做幂等保护 | 避免重试、调度补偿或人工操作导致重复考勤数据发送 |
| 首版不引入账号体系 | 符合内网单管理员场景，减少非核心功能，践行 YAGNI |
| 首版仅支持单一外部上报目标 | 先收敛接口配置与投递逻辑，降低系统复杂度 |

## Issues Encountered
| Issue | Resolution |
|-------|------------|
| 仓库为空且未初始化 git | 作为项目初始化场景处理，不假设已有工程规范 |
| 第一次补丁更新 findings 失败 | 重新读取当前文件内容后按精确上下文重试 |

## Resources
- `/Users/wesleyw/Project/syslog`
- `/Users/wesleyw/.codex/superpowers/skills/brainstorming/SKILL.md`
- `/Users/wesleyw/.codex/skills/planning-with-files/skills/planning-with-files/SKILL.md`
- `/Users/wesleyw/.codex/superpowers/skills/test-driven-development/SKILL.md`

## Visual/Browser Findings
- 暂无

## Feishu Record ID Query Fix (2026-03-21)
- 飞书 `attendance/v1/user_flows/batch_create` 在当前租户实际返回 `code=0` 和导入回显字段，但不会直接回传 `record_id`；把“HTTP 200 + code 0 + 无 record_id”视为失败会导致系统错误重试。
- `record_id` 需要通过 `attendance/v1/user_tasks/query` 再查一次当日打卡结果获得，查询维度至少要带 `employee_type=employee_id`、`user_ids`、`check_date_from/check_date_to`。
- 在当前实现里，最稳妥的匹配条件是 `user_id / creator_id / location_name / check_time / comment`；匹配成功后优先取 `check_in_record.record_id` 或 `check_out_record.record_id`，若嵌套对象没有则退回 `check_in_record_id / check_out_record_id`。
- 使用 `time.Local` 将秒级 `check_time` 映射为 `yyyyMMdd` 查询日，可以和当前服务时区 `Asia/Shanghai` 保持一致，避免 UTC 日期与企业工作日错位。

## Feishu Success Notification Findings (2026-03-22)
- 飞书打卡导入成功后给员工发通知，最稳妥的做法是复用同一套 `tenant_access_token` 与 Feishu HTTP client，而不是新增独立消息客户端，符合 KISS/DRY。
- 飞书消息接口可直接使用 `receive_id_type=user_id`；在当前系统里，员工档案已有 `feishu_employee_id`，无需再新增 open_id/union_id 字段，符合 YAGNI。
- 打卡导入成功与消息通知成功是两个不同事务：如果把消息失败视为整条考勤失败，会导致已导入飞书考勤的数据被本地错误重试，因此通知状态必须与 `report_status` 解耦。
- `attendance_reports` 已经是现成的“异步外部交互状态载体”，为其增加通知状态字段比再建一张通知表更简单，也更符合当前单通知场景的 KISS/YAGNI 边界。
- `clear` 语义只负责删除或清空远端打卡，不应向员工发送“打卡成功”通知，否则业务语义混乱，因此将其通知状态固定为 `skipped`。
- 为避免 Feishu IM 重试造成重复单聊消息，消息请求应使用稳定去重键；基于 `report.idempotency_key` 派生稳定 `uuid` 能满足 1 小时内的飞书去重要求，同时不引入额外存储。
- 手工调试发送如果绕开 dispatcher 单独发消息，会造成正式链路与调试链路分叉；继续走 dispatcher 可以保持一套行为、一套日志和一套重试策略，符合 DRY/SRP。

## Real Data Cleanup & Page Structure Findings (2026-03-22)
- 当前前端真实数据层已经接到 `/api`，残留的“mock 感”主要不在接口，而在运行时壳层和页面摘要：
  - `AppShell` 里仍有硬编码 `commandSignals`、`pulseNotes`、`quickActions`
  - 这些值会让系统在真实数据为空或异常时仍表现得像“稳定运行”，属于误导性的伪数据
- `DashboardPage`、`AttendancePage`、`LogsPage` 都已经有真实数据，但页面结构仍偏“单面板堆叠”，下一步应把“概览 / 待处理 / 明细”真正分层，而不是继续堆文案。
- `EmployeesPage` 当前右侧卡片列表信息重复，设备信息以长字符串拼接呈现，不利于扫描；更适合改为紧凑名册表格或表单-列表双栏的密集布局。
- `SettingsPage` 当前虽然接了真实数据，但“配置说明”仍是大段静态说明，缺少对当前配置状态的摘要；同时 `SettingsForm` 的时间校验只验证形状，不验证合法时间范围。
- `DebugPage` 是真实链路，但反馈区域仍是单行状态文案，不适合作为运维调试页面；应拆成“动作区”和“结果区”。
- 运行时代码之外，还存在两处“历史 mock/演示痕迹”会持续误导使用者：
  - `README.md` 仍宣称 UDP handler 是占位实现、前端仍依赖 mock 数据，这与当前系统状态冲突
  - `system_settings` 迁移里仍插入 `report_target_url`、`feishu_creator_employee_id` 两个已废弃键，会把历史兼容字段继续带进数据库
- 仓库缺少正式的数据清理入口；对于已经写入 MySQL 的演示员工、演示日志和演示考勤，只能靠手工 SQL 清理，容易误删真实配置。

## Configurable Syslog Receive Rule Findings (2026-03-22)
- 当前 parser 早已只识别 `connect` / `disconnect`，真正导致“日志很杂”的不是 parser，而是 pipeline 把所有未识别 syslog 也保存成了 `syslog_messages.parse_status=failed`。
- 对这个场景，继续往通用 `system_settings` 里塞 JSON 不合适：用户需要的是“规则列表 + 新增/编辑/删除 + 启停”，这天然是独立资源，单独建 `syslog_receive_rules` 表和 CRUD API 更符合 SRP。
- “用户输入 schema”在当前系统里收敛成“正则 + 命名分组映射到现有字段”最稳妥：
  - 足够满足 connect/disconnect 提取
  - 不需要引入通用 DSL/解释器
  - 不会破坏当前 `client_events` 固定列模型
- 未命中规则的 syslog 直接丢弃后，日志页会自然只剩有效 connect/disconnect 记录；这比在 UI 侧二次过滤更干净，也避免无意义 failed 记录持续占用保留窗口。
- 规则校验必须前置到 service 层，而不是等运行时出错：
  - `eventType` 只能是 `connect` 或 `disconnect`
  - `messagePattern` 必须可编译
  - `stationMacGroup` 必填
  - 配置的分组名必须真实存在于正则的命名分组中
  - `eventTimeGroup` 设置时必须同时提供 `eventTimeLayout`
- Debug syslog 注入也必须复用同一套规则匹配逻辑；否则设置页里改了规则，调试页看到的 parseStatus 和真实入库行为会分叉。

## Syslog Rule Ordering & Raw Inbox Findings (2026-03-22)
- 当前 `SyslogPipeline` 在规则模式下会直接丢弃未命中 syslog，不再落 `syslog_messages`；这使得日志面板干净，但丢掉了接收审计能力。
- 最小正确修复不是恢复“全部都算 failed”，而是把原始接收与有效事件拆开：`syslog_messages` 保存所有接收，`client_events` 只保存命中规则的结构化事件。
- 为了不打乱现有日志页用途，`/api/logs` 需要新增 scope；默认仍看有效事件，额外提供“全部接收”视图。
- 规则排序最简单稳定的实现是显式 `sort_order` 字段，而不是按 `id` 或更新时间隐式排序。
- 规则命中预览应复用同一套 matcher，避免前端和后端各自实现一份正则提取逻辑。

## Syslog Rule Ordering & Raw Inbox Findings (continued)
- 规则顺序通过 `sort_order` 显式持久化，匹配与列表都统一按该字段排序，避免用 `id` 假装优先级。
- 规则命中预览通过后端 `PreviewRule` 直接复用生产 `applySyslogRule`，没有引入第二套前端解析逻辑。
- 原始接收与有效事件已拆层：未命中的 syslog 会以 `parse_status=ignored` 保存在 `syslog_messages`，但不会进入 `client_events`、考勤和飞书链路。
- 日志页仍以“有效事件”为默认视图，`scope=all` 仅作为排查入口，不会打乱原有工作台语义。
