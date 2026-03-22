# Task Plan: Syslog Server Initialization

## Active Execution Addendum: Migration Hotfix for `sort_order`

### Execution Goal
修复旧数据库启动时 `syslog_receive_rules.sort_order` 缺失导致的 migration 失败，确保单文件 migration 在历史库上先补列再写入默认规则。

### Execution Phase
Phase M2

### Execution Phases
#### Phase M1: Root Cause & Regression Test
- [x] 复现并确认 `Unknown column 'sort_order' in 'field list'` 发生在 migration 阶段
- [x] 锁定 `001_init.sql` 中 seed insert 先于 `sort_order` 补列的顺序错误
- [x] 先补回归测试，确保修复前失败
- **Status:** completed

#### Phase M2: Migration Fix & Verification
- [x] 调整 `001_init.sql` 顺序，先补 `sort_order` 再执行默认规则 seed
- [x] 运行 bootstrap 定向测试确认回归已消失
- [x] 运行 backend 全量测试确认未引入其他迁移副作用
- **Status:** completed

## Active Execution Addendum: Frontend Debug Tools

### Execution Goal
新增一个独立前端调试页，支持手工注入 syslog 报文与手工把单条考勤记录按上班/下班维度发送到飞书；后端通过受控 debug API 复用正式 `SyslogPipeline` 和单条 `AttendanceReportDispatcher` 链路，避免旁路逻辑。

### Execution Phase
Phase D4

### Execution Phases
#### Phase D1: Contract & Failing Tests
- [x] 盘点现有 router、attendance/logs 页面与 dispatcher 能力
- [x] 先补外层失败测试，锁定 `/debug` 前端入口与 `/api/debug/*` 后端路由
- [x] 明确“单条 dispatch，不扫描整队列”的行为边界
- **Status:** completed

#### Phase D2: Backend Debug API & Service
- [x] 新增 `DebugAdminService`，收口 syslog 注入与单条飞书 dispatch 语义
- [x] 为 dispatcher 增加单条 `DispatchReport` 入口，避免手工按钮顺带跑整批 pending/failed 队列
- [x] 新增 debug handlers 和 router 注入，保持 handler 只做解码和状态映射
- **Status:** completed

#### Phase D3: Frontend Debug Page
- [x] 新增 `/debug` 路由与导航入口
- [x] 实现 syslog 注入表单、飞书单条上报表格和确认交互
- [x] 扩展前端 API 类型与调用函数，保持现有页面不受影响
- **Status:** completed

#### Phase D4: Verification & Wrap-up
- [x] 运行后端全量测试
- [x] 运行前端全量测试
- [x] 运行前端生产构建并回写规划文件
- **Status:** completed

## Active Execution Addendum: Feishu Attendance Reporting

### Execution Goal
在现有考勤确认链路上接入飞书人事打卡上报：系统确认上班/下班后异步导入飞书打卡流水，人工修正/清除时支持删旧再导新；同时扩展员工档案与系统设置，维护飞书员工 ID 与飞书应用凭据，保持现有考勤计算和管理 API 语义稳定。

### Execution Phase
Phase L1

### Execution Phases
#### Phase L1: Contract & Failing Tests
- [x] 盘点现有 report queue、员工字段、设置字段与启动链路
- [x] 先补后端失败测试，锁定飞书 settings、员工飞书 ID、dispatcher 语义与 correction 的删旧再导新逻辑
- [x] 先补前端失败测试，锁定员工/设置页面的新字段与提交 payload
- **Status:** completed

#### Phase L2: Backend Domain & Persistence
- [x] 扩展 migration、domain、repository，新增飞书字段和 report 外部流水跟踪字段
- [x] 改造 report 生成逻辑与 correction 逻辑，复用现有队列而不是另起流程
- [x] 保持 handler / service 接口职责清晰，避免 report 细节泄露到无关层
- **Status:** completed

#### Phase L3: Feishu Dispatcher
- [x] 实现 tenant_access_token 缓存与飞书 attendance client
- [x] 实现后台 dispatcher，处理 pending/failed report 的删除、导入、重试与状态落库
- [x] 将 dispatcher 接入 server 启动与优雅退出
- **Status:** completed

#### Phase L4: Frontend Integration
- [x] 扩展 employee/settings API 类型与页面表单
- [x] 移除旧 report target URL 的 UI，替换为飞书 app/creator/location 配置
- [x] 补齐新字段的表单校验与保存行为测试
- **Status:** completed

#### Phase L5: Verification & Wrap-up
- [x] 运行后端相关 go test
- [x] 运行前端测试与 build
- [x] 回写 findings/progress，记录残余风险与后续项
- **Status:** completed

## Active Execution Addendum: Feishu Record ID Query Fix

### Execution Goal
修正飞书打卡导入后的 `record_id` 获取语义：`batch_create` 成功后不再要求响应体直接回传 `record_id`，而是追加调用 `attendance/v1/user_tasks/query` 回查匹配的打卡结果并提取 `record_id`，以保持后续 `batch_del` 能力可用。

### Execution Phase
Phase FQ2

### Execution Phases
#### Phase FQ1: Root Cause & Contract Lock
- [x] 根据真实飞书响应与补充文档确认 `record_id` 不会由 `batch_create` 直接回传
- [x] 先补 HTTP client 失败测试，锁定“导入成功后通过 `user_tasks/query` 回查 `record_id`”行为
- **Status:** completed

#### Phase FQ2: Client Fix & Verification
- [x] 调整 `FeishuAttendanceHTTPClient.CreateFlow`，在 create 成功后回查 `user_tasks/query`
- [x] 通过匹配 `user_id / creator_id / location_name / check_time / comment` 提取对应打卡记录 ID
- [x] 运行 service 与 backend 全量测试验证调度链未回归
- **Status:** completed

## Active Execution Addendum: Frontend Command Deck UI Redesign

### Execution Goal
按已确认方案完成前端 UI 全量重设计：`Command Deck` 信息架构、`Cold Slate` 白天/夜间双主题、中文主文案、共享 UI 组件层、五个业务页统一视觉与交互语义，并保持现有后端 API 契约与关键行为不变。

### Execution Phase
Phase U1

### Execution Phases
#### Phase U1: Baseline & Test Design
- [x] 盘点当前前端结构、已有脏工作区与测试约束
- [x] 先写或改写会被这次重构影响的前端测试
- [x] 明确允许保持不变的 API/行为边界
- **Status:** completed

#### Phase U2: Theme & Shell Foundation
- [x] 建立双主题设计令牌与全局布局样式
- [x] 重构 AppShell、导航、主题切换与中文文案
- [x] 抽出共享 UI 组件或共享 class 语义，消除核心内联样式
- **Status:** completed

#### Phase U3: Page Redesign
- [x] 重做 Dashboard / Logs / Attendance / Employees / Settings 五页布局
- [x] 保留现有业务行为，统一状态标签、表格、表单和空态
- [x] 更新对应测试断言与可访问性语义
- **Status:** completed

#### Phase U4: Verification & Cleanup
- [x] 运行前端测试并修正失败
- [x] 运行前端构建验证主题与样式改造未破坏编译
- [x] 复查规划文件并记录最终结果、风险与后续项
- **Status:** completed

## Active Execution Addendum: Task 13 Frontend API Integration

### Execution Goal
将前端从 mock 数据层切换到真实后端 HTTP API，覆盖员工管理、系统设置、考勤查询与人工修正，并补齐真实契约、加载保护与待处理语义的一致性。

### Execution Phase
Phase F1

### Execution Phases
#### Phase F1: API Contract Discovery
- [x] 盘点前端现有 mock 数据入口与页面依赖
- [x] 对齐后端真实接口请求/响应契约
- [x] 识别需要调整的前端状态与错误处理
- **Status:** completed

#### Phase F2: Data Layer Wiring
- [x] 将前端查询/写入改接真实 HTTP API
- [x] 统一处理加载、错误、空态和表单提交反馈
- [x] 保持现有页面职责和结构稳定
- **Status:** completed

#### Phase F3: Verification
- [x] 更新或补充前端测试
- [x] 运行前端测试与构建
- [x] 做一次端到端联调检查
- **Status:** completed

### Task 13 Outcome
- 已完成从 mock API 到真实后端 `/api` 的切换，覆盖 employees / attendance / settings / logs / dashboard。
- 已补齐前端对后端 numeric ID 契约的归一化，避免依赖运行时隐式类型转换。
- 已修复 settings 首次加载未完成或失败时的默认值覆盖风险。
- 已统一 AttendanceTable 与 Dashboard 的待处理原因语义，并为 pending clock-in / manual resolved 等边界补充测试。
- 最终状态通过 spec review 与 quality review。

## Completed Addendum: Task 12 Code Quality Review

### Review Goal
基于基线 `ad1e707eb488b8961c6f6e4951a985853f23d37b` 对当前 `HEAD` 的 Task 12 实现做代码质量复核，重点覆盖：
- 员工 + devices 事务一致性与更新语义
- 人工修正 patch 语义、`version/status` 变化、report 生成与 `target_url`
- settings 批量更新约束
- handler 请求/响应契约和错误处理
- 测试是否充分锁定本轮行为

### Review Phase
Phase R1

### Review Phases
#### Phase R1: Scope & Diff Discovery
- [x] 确认 review 基线与关注文件
- [x] 检查现有规划文件，避免旧任务上下文污染
- [x] 建立本轮 review 记录
- **Status:** completed

#### Phase R2: Implementation Review
- [x] 审查 service / repository / handler 的实现语义
- [x] 核对事务边界、状态流转与领域约束
- [x] 记录所有潜在问题及影响
- **Status:** completed

#### Phase R3: Tests & Verification
- [x] 审查新增或更新测试是否覆盖关键路径
- [x] 运行必要的定向测试或静态验证
- [x] 记录未验证项与残余风险
- **Status:** completed

#### Phase R4: Reporting
- [x] 输出 Strengths / Issues / Recommendations / Assessment
- [x] 明确是否可合并
- **Status:** completed

## Active Review Addendum: Task 12 Fix Review

### Review Goal
基于基线 `1206423b0b5bd4c5314eed6cd56808ea5176700c` 复核目标提交 `ac998e28cf1d4f5a2f395aab5521422dbfe27aa5`，重点验证上轮提出的问题是否被真正修复，并找出新的质量风险：
- attendance correction 的 no-op / clear-null / status / idempotency / transaction 语义
- employee write 的 not-found、返回体真实性、事务与错误处理
- handler 的 400/404/409/500 映射与 trailing JSON 拒绝逻辑
- tests 是否真正锁住关键边界而不只是 happy path

### Review Phase
Phase F1

### Review Phases
#### Phase F1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与文件范围
- [x] 读取上轮 review 结论作为对照项
- [x] 建立本轮 fix review 记录
- **Status:** completed

#### Phase F2: Implementation Review
- [x] 审查 service / repository / handler / report 相关净变更
- [x] 对照上轮问题逐项验证修复是否成立
- [x] 记录新增风险与回归点
- **Status:** completed

#### Phase F3: Tests & Verification
- [x] 审查新增测试是否锁定关键边界
- [x] 运行必要定向测试
- [x] 记录未覆盖项与残余风险
- **Status:** completed

#### Phase F4: Reporting
- [ ] 输出 Strengths / Issues / Recommendations / Assessment
- [ ] 明确是否可合并
- **Status:** completed

## Active Review Addendum: Task 12 Follow-up Review

### Review Goal
基于基线 `ac998e28cf1d4f5a2f395aab5521422dbfe27aa5` 复核目标提交 `c703ecd48da8060f480eb2ad8f464b9b4624e140`，重点验证：
- 上轮 Critical 的 `RowsAffected==0` 误判是否已解除
- employee 主字段不变但 devices 变化的更新是否正确
- disable 已 disabled 员工是否不再误报 404
- 新的存在性预检是否引入明显问题
- attendance correction handler 的 404 覆盖是否足够
- duplicate key 409 判定是否更稳

### Review Phase
Phase G1

### Review Phases
#### Phase G1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与文件范围
- [x] 读取上轮 fix review 结论作为对照项
- [x] 建立本轮 follow-up review 记录
- **Status:** completed

#### Phase G2: Implementation Review
- [x] 审查 `admin_services` / `employee_repository` / `response` 的后续修复逻辑
- [x] 逐项验证 6 个重点问题
- [x] 记录新增风险与回归点
- **Status:** completed

#### Phase G3: Tests & Verification
- [x] 审查新增测试是否锁住关键边界
- [x] 运行必要定向测试
- [x] 记录未覆盖项与残余风险
- **Status:** completed

#### Phase G4: Reporting
- [ ] 输出 Strengths / Issues / Recommendations / Assessment
- [ ] 明确是否可合并
- **Status:** in_progress

## Active Review Addendum: Task 13 Frontend Review

### Review Goal
基于基线 `c703ecd48da8060f480eb2ad8f464b9b4624e140` 复核目标提交 `d2e8919e8af3d323d1d664701613f28dac1d1aea`，重点验证：
- `lib/api.ts` 的真实契约处理，尤其是 settings/logs 对后端大写字段的归一化是否稳
- 页面层面的状态同步、竞态、重复请求与错误处理
- attendance manual correction 的 draft / join 逻辑是否有行为漏洞
- settings form 改成提交时收集值后的重置、验证与可维护性
- router 测试、fetch mock 和页面测试是否真正锁住关键行为
- 是否还有明显 UX / 代码质量问题应在本轮修掉

### Review Phase
Phase H4

### Review Phases
#### Phase H1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与净变更文件
- [x] 更新计划文件并登记本轮关注点
- [x] 建立 Task 13 review 记录
- **Status:** completed

#### Phase H2: Implementation Review
- [x] 审查 `api.ts`、attendance/employees/settings/dashboard/logs 页面实现
- [x] 对照后端 JSON 契约检查归一化与真实请求/响应处理
- [x] 记录状态同步、竞态和 UX 风险
- **Status:** completed

#### Phase H3: Tests & Verification
- [x] 审查 router/page tests 与 `fetchMock.ts`
- [x] 核对测试是否锁定真实契约和危险边界
- [x] 记录未覆盖项与 mock 假设
- **Status:** completed

#### Phase H4: Reporting
- [ ] 输出 Strengths / Issues / Recommendations / Assessment
- [ ] 明确是否可合并
- **Status:** in_progress

## Active Review Addendum: Frontend Follow-up Review

### Review Goal
基于基线 `d2e8919e8af3d323d1d664701613f28dac1d1aea` 复核目标提交 `b0a65c2e19f886bdfa70747d848d75044afd87ae`，重点验证：
- `frontend/src/lib/api.ts` 对后端真实 JSON 契约的适配与容错
- `AttendancePage` / `AttendanceTable` 的修正草稿与状态逻辑
- `Dashboard` / `Logs` / `Settings` 的运行时数据流是否干净
- 测试是否真正覆盖关键行为，以及是否存在脆弱断言

### Review Phase
Phase I4

### Review Phases
#### Phase I1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与实际净变更文件
- [x] 识别本轮主要变更集中在 attendance/dashboard tests 与状态判定
- **Status:** completed

#### Phase I2: Implementation Review
- [x] 审查当前 `api.ts`、attendance/dashboard/settings 运行时逻辑
- [x] 核对本轮状态语义改动是否与后端真实语义一致
- [x] 记录残留与新增问题
- **Status:** completed

#### Phase I3: Tests & Verification
- [x] 审查更新后的 attendance/dashboard tests
- [x] 检查 logs/settings/fetch mock 是否补足关键边界
- [x] 记录 mock 假设与脆弱断言
- **Status:** completed

#### Phase I4: Reporting
- [ ] 仅输出实际问题并按严重度排序
- [ ] 明确是否可合并
- **Status:** in_progress

## Active Review Addendum: Frontend Final Fix Review

### Review Goal
基于基线 `b0a65c2e19f886bdfa70747d848d75044afd87ae` 复核目标提交 `6003b14dcdadb6be8593cddaddb85f51314bb120`，重点验证：
- Settings 首次加载完成前不可提交
- employees / attendance 的 numeric id 已在 API 层归一化为字符串
- manual 但已恢复正常的考勤记录不再算待处理，且 `AttendanceTable` 与 `Dashboard` 语义一致
- 没有明显范围漂移

### Review Phase
Phase J4

### Review Phases
#### Phase J1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与净变更文件
- [x] 判断是否存在明显范围漂移
- **Status:** completed

#### Phase J2: Implementation Review
- [x] 审查 settings 禁提交流程
- [x] 审查 `api.ts` 的 employees / attendance ID 归一化
- [x] 审查 attendance/dashboard attention 语义是否统一
- **Status:** completed

#### Phase J3: Verification
- [x] 运行前端测试
- [x] 运行前端构建
- [x] 记录残余风险或确认通过
- **Status:** completed

#### Phase J4: Reporting
- [x] 若无问题则仅输出 `Spec compliance: pass`
- **Status:** completed

## Active Review Addendum: Frontend Commit 6003b14 Re-review

### Review Goal
基于基线 `b0a65c2e19f886bdfa70747d848d75044afd87ae` 重新复核目标提交 `6003b14dcdadb6be8593cddaddb85f51314bb120`，重点验证：
- `frontend/src/lib/api.ts` 的归一化是否稳健且不再依赖隐式类型转换
- `SettingsPage` / `SettingsForm` 的加载禁用与保存路径是否真正闭合
- `AttendanceTable` 与 `Dashboard` 共用的待处理语义是否一致且不会误判已恢复正常或“异常列为 none”的记录
- 新增测试是否真正锁住这些边界

### Review Phase
Phase K4

### Review Phases
#### Phase K1: Scope & Diff Discovery
- [x] 确认 review 基线、目标提交与本轮关注点
- [x] 识别本轮变更集中在 `api.ts`、settings、attendance/dashboard 与测试
- **Status:** completed

#### Phase K2: Implementation Review
- [x] 审查 `api.ts` 归一化实现
- [x] 审查 settings 加载失败后的保存路径
- [x] 审查 attention helper 与前端展示语义是否一致
- **Status:** completed

#### Phase K3: Tests & Verification
- [x] 审查新增 `api.test.ts`、settings/attendance/dashboard tests
- [x] 识别未覆盖的关键边界
- **Status:** completed

#### Phase K4: Reporting
- [ ] 仅输出实际问题并按严重度排序
- [ ] 明确是否可合并
- **Status:** in_progress

## Goal
初始化一个可运行的 syslog 服务端项目，包含 React 前端、Golang 后端和 MySQL 数据存储，并覆盖 AP syslog 消息接收、事件处理、消息存储、考勤汇总与外部 API 上报能力。

## Current Phase
Phase 1

## Phases
### Phase 1: Requirements & Discovery
- [x] 理解用户目标与技术栈
- [ ] 明确 syslog 协议范围、业务消息流与考勤判断规则
- [x] 记录当前仓库状态与限制
- **Status:** in_progress

### Phase 2: Design & Plan
- [ ] 产出系统设计方案与技术边界
- [ ] 获得用户对方案的确认
- [ ] 拆解初始化步骤与验证方式
- **Status:** completed

### Phase 3: Project Bootstrap
- [ ] 初始化仓库与基础目录结构
- [ ] 初始化 React 前端、Go 后端、数据库脚本与本地运行编排
- [ ] 建立统一配置与开发说明
- **Status:** completed

### Phase 4: Core Features
- [ ] 实现消息接收入口
- [ ] 实现消息处理流水线
- [ ] 实现 MySQL 持久化
- [ ] 实现考勤汇总与外部 API 发送能力
- **Status:** completed

### Phase 5: Testing & Verification
- [ ] 为核心能力补充测试
- [ ] 运行关键验证命令
- [ ] 修正发现的问题
- **Status:** pending

### Phase 6: Delivery
- [ ] 汇总改动与设计决策
- [ ] 说明运行方式、限制与后续建议
- [ ] 完成交付
- **Status:** pending

## Key Questions
1. 原始 syslog 的保留天数是否需要前端可配置？
2. 是否需要支持手工触发“重新汇总并重发某一天”的补偿操作？
3. 设计阶段是否需要把原始日志保留天数默认值先定为 30 天？

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| 使用文件化计划推进整个初始化任务 | 任务跨前后端与数据库，步骤多，便于持续记录与恢复上下文 |
| 先做需求澄清与方案确认，再进入实现 | 符合 KISS/YAGNI，避免在协议范围和 MVP 边界不清时过度设计 |
| “发送消息”模块按业务上报设计，而不是 syslog 转发 | 用户明确说明输出是基于分析结果调用外部 API 发送其他类型消息 |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| 当前目录不是 git 仓库 | 1 | 记录现状，后续在实现阶段决定是否初始化 git |
| `findings.md` 首次补丁未命中上下文 | 1 | 读取当前文件后按精确上下文重新补丁 |

## Notes
- 任何实现前先确认设计方案
- 每完成一个阶段都更新状态与验证记录
- 外部资料与不可信内容只写入 findings.md

## Session: 2026-03-22 Feishu Success Notification

### Phase FN1: Scope & Notification Semantics
- [x] 明确通知仅在 `clock_in` / `clock_out` 飞书打卡成功后发送
- [x] 明确 `clear` 不通知，通知失败不影响打卡成功状态
- [x] 明确接收者直接使用员工 `feishu_employee_id` 作为 `user_id`
- **Status:** completed

### Phase FN2: Backend Persistence & Dispatcher
- [x] 为 `attendance_reports` 增加通知状态、响应与重试字段
- [x] 扩展 Feishu client 支持 `im/v1/messages` 文本消息发送
- [x] 在 dispatcher 中实现“打卡成功后通知”与“通知失败单独补偿重试”
- **Status:** completed

### Phase FN3: Frontend Debug Visibility
- [x] 扩展调试发送响应模型，展示通知状态、消息 ID 与失败摘要
- [x] 确认手工调试发送也复用正式通知链路
- **Status:** completed

### Phase FN4: Verification & Delivery
- [x] 运行 backend 全量测试
- [x] 运行 frontend 全量测试与构建
- [x] 汇总交付说明与边界
- **Status:** completed

## Session: 2026-03-22 Real Data Cleanup & Page Structure Review

### Phase RS1: Discovery & Structure Audit
- [x] 盘点前端运行时代码中的硬编码信号、伪摘要与页面结构问题
- [x] 区分测试 mock 与运行时 mock，避免误删测试桩
- [x] 明确每页需要收口的结构问题和真实数据替换范围
- **Status:** completed

### Phase RS2: Test Lock
- [x] 先补前端行为测试，锁定真实壳层摘要、页面分层与关键表单边界
- [x] 验证新增测试先失败，确保覆盖的是新行为而不是既有行为
- **Status:** completed

### Phase RS3: Implementation
- [x] 将 AppShell 的硬编码摘要改为真实数据推导
- [x] 调整 Dashboard / Logs / Attendance / Employees / Settings / Debug 六页结构
- [x] 消除仍会误导用户的运行时伪数据文案
- **Status:** completed

### Phase RS4: Verification & Wrap-up
- [x] 运行前端全量测试与构建
- [x] 运行必要的后端测试，确认接口契约未回归
- [x] 回写规划文件并汇总剩余风险
- **Status:** completed

## Session: 2026-03-22 Configurable Syslog Receive Rules

### Phase SR1: Contract & Red Tests
- [x] 明确采用“字段映射规则”而不是自由 JSON schema
- [x] 明确未命中规则的 syslog 直接丢弃，不再写 failed 原始日志
- [x] 先补后端与前端失败测试，锁定规则 CRUD、设置页规则工作区和未命中丢弃语义
- **Status:** completed

### Phase SR2: Backend Rule Engine & API
- [x] 新增 `syslog_receive_rules` 表与默认 connect/disconnect 规则
- [x] 实现规则仓储、admin service、handler 与 router 注入
- [x] 将 `SyslogPipeline` 改为按启用规则匹配后再入库
- **Status:** completed

### Phase SR3: Frontend Settings Integration
- [x] 扩展前端 API 类型与 `/api/syslog-rules` 调用
- [x] 在设置页新增 `Syslog 接收规则` 工作区，支持新建、编辑、删除
- [x] 明确界面文案：仅接收 `connect / disconnect`，未命中直接丢弃
- **Status:** completed

### Phase SR4: Verification & Delivery
- [x] 运行 backend 全量测试
- [x] 运行 frontend 全量测试
- [x] 运行 frontend 生产构建
- [x] 回写规划文件
- **Status:** completed

## Active Execution Addendum: Syslog Rule Ordering & Raw Inbox

### Execution Goal
为 Syslog 接收规则补充可调整顺序与规则命中预览能力，并新增可查看全部接收原始数据的日志视图；保持当前有效事件视图、考勤链路与飞书链路语义稳定。

### Execution Phase
Phase SR4

### Execution Phases
#### Phase SR1: Contract & Failing Tests
- [x] 锁定规则排序、预览和全部接收视图的契约
- [x] 先补后端失败测试，覆盖规则顺序、预览、未命中数据持久化与日志 scope
- [x] 先补前端失败测试，覆盖规则移动、预览交互与日志 scope 切换
- **Status:** in_progress

#### Phase SR2: Backend Rule Ordering & Raw Inbox
- [x] 为规则增加排序与预览接口
- [x] 调整 pipeline，保存所有接收原始日志，并为命中规则记录元信息
- [x] 扩展日志查询，支持 matched/all 两种视图
- **Status:** pending

#### Phase SR3: Frontend Settings & Logs
- [x] 为设置页规则工作区增加上移下移与命中预览
- [x] 为日志页增加有效事件/全部接收切换
- [x] 保持详情弹层与现有筛选、分页语义一致
- **Status:** pending

#### Phase SR4: Verification & Wrap-up
- [x] 运行后端测试
- [x] 运行前端测试与 build
- [x] 回写 findings/progress，记录数据保留与视图边界
- **Status:** pending
