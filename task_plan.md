# Task Plan: Syslog Server Initialization

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
- **Status:** pending

### Phase 3: Project Bootstrap
- [ ] 初始化仓库与基础目录结构
- [ ] 初始化 React 前端、Go 后端、数据库脚本与本地运行编排
- [ ] 建立统一配置与开发说明
- **Status:** pending

### Phase 4: Core Features
- [ ] 实现消息接收入口
- [ ] 实现消息处理流水线
- [ ] 实现 MySQL 持久化
- [ ] 实现考勤汇总与外部 API 发送能力
- **Status:** pending

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
