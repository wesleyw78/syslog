# Syslog Attendance System Design

## Overview

本项目目标是初始化一个面向 AP syslog 日志的考勤处理系统，技术栈为 React + Golang + MySQL。

系统核心能力包括：

- 接收 AP 通过 `UDP 514` 发送的 syslog 消息
- 解析客户端连接与断开事件
- 基于员工与设备 MAC 映射计算每日考勤
- 将考勤结果通过可配置的 `HTTP JSON` 接口上报到单一外部服务器
- 提供前端控制台用于运维监控、资料维护、考勤查看和人工修正

系统统一按 `Asia/Shanghai` 时区处理当天归属、调度和上报时间。

## Scope

首版范围如下：

- 支持 `RFC 3164 + RFC 5424` 的 syslog 基础兼容与自动识别
- 当前 AP 日志格式按半结构化文本解析，至少支持以下事件特征：
  - `connect`
  - `disconnect`
  - `Station[...]` 客户端 MAC
- 员工主数据由前端手工维护：
  - 工号
  - 系统编号
  - 姓名
  - 多个设备 MAC
- 上班规则：
  - 员工当天首次连接立即认定为上班
  - 立即上报外部 API
- 下班规则：
  - 员工当天最后一次断开作为下班候选时间
  - 到前端配置的日终时间后再确认并上报
  - 如果当天无断开事件，则不自动上报，留待人工处理
- 首版支持管理员手工修正上下班时间，并允许修正后重新上报
- 原始 syslog 只保留固定天数，不做永久保存
- 首版不做登录鉴权，按内网单管理员工具实现

## Architecture

采用“领域分层单体”方案：一个 Go 后端服务承载所有后端能力，但内部按职责拆分模块。

### 1. Syslog Ingest

职责：

- 监听 `UDP 514`
- 接收原始 syslog 消息
- 写入原始消息存储

设计原则：

- 只负责接入，不承担业务判断，符合 SRP

### 2. Parser

职责：

- 将原始 syslog 解析为统一事件结构
- 提取时间、事件类型、客户端 MAC、AP MAC、SSID、IP、hostname 等字段

设计原则：

- 将协议与业务分离，后续扩展新的 AP 日志格式时只扩展解析器，符合 OCP

### 3. Attendance Processor

职责：

- 基于解析后的连接/断开事件和员工设备映射，按“员工 + 日期”聚合考勤
- 维护上班、下班、异常、人工修正状态

设计原则：

- 业务规则集中管理，避免散落在控制器或存储层，符合 KISS 与 SRP

### 4. Reporter

职责：

- 将考勤结果封装为可配置的 `HTTP JSON`
- 携带幂等键调用单一外部服务器
- 记录请求与响应结果

设计原则：

- 上报协议独立，后续可替换为真实接口适配器，符合 DIP/OCP

### 5. Scheduler

职责：

- 按 `Asia/Shanghai` 时区执行日终任务
- 触发下班确认与上报
- 清理超过保留天数的原始 syslog

设计原则：

- 将时间驱动任务与请求链路解耦

### 6. Admin API + React Console

职责：

- 提供前端控制台 API
- 支持日志查看、员工资料维护、考勤查看与修正、系统配置管理

设计原则：

- 前端聚焦运维与业务闭环，不引入非核心账号体系，符合 YAGNI

## Data Flow

系统主链路如下：

`AP syslog -> ingest -> parser -> event storage -> attendance processor -> attendance record -> reporter -> external API`

### 1. Syslog Receive

- Go 后端监听 `UDP 514`
- 每收到一条消息，先写入 `syslog_messages`
- 成功解析后写入 `client_events`
- 解析失败则保留原文并标记失败状态

### 2. Event Match

- 通过 `station_mac` 到 `employee_devices` 查找员工
- 匹配成功则记录 `matched_employee_id`
- 匹配失败则标记异常并在前端展示

### 3. Attendance Aggregate

按 `employee_id + attendance_date` 聚合：

- `connect`
  - 若当天没有 `first_connect_at`，写入上班时间
  - 立即生成上班上报任务
  - 若已有上班时间，则忽略更晚连接
- `disconnect`
  - 更新 `last_disconnect_at`
  - 不立即上报
  - 始终保留当天最晚断开时间，等待日终确认

### 4. Day-End Finalization

每天到配置的日终时间：

- 若存在 `last_disconnect_at`
  - 生成下班上报任务
  - 上报成功后标记下班已上报
- 若不存在 `last_disconnect_at`
  - 标记 `missing_disconnect`
  - 不自动上报
  - 等待人工处理

### 5. Manual Correction

管理员可在前端手工修正上下班时间：

- 更新 `attendance_records`
- 标记 `source_mode=manual`
- 重新生成上报任务
- 使用新的幂等键再次上报

## Attendance Rules

### Aggregation Dimension

- 一个员工可绑定多个设备 MAC
- 当天考勤按员工维度聚合
- 上班取所有设备中的最早连接时间
- 下班取所有设备中的最晚断开时间

### Clock-In Rule

- 员工当天首次连接即视为上班
- 立即调用外部 API 上报

### Clock-Out Rule

- 员工当天最后断开作为下班候选时间
- 到日终时间才确认
- 如果没有断开事件，不自动生成下班时间

### Exception Rule

首版至少包含以下异常：

- `parse_failed`
- `unmatched_device`
- `missing_disconnect`
- `report_failed`
- `manual_corrected`

## Database Model

### `employees`

字段建议：

- `id`
- `employee_no`
- `system_no`
- `name`
- `status`
- `created_at`
- `updated_at`

### `employee_devices`

字段建议：

- `id`
- `employee_id`
- `mac_address`
- `device_label`
- `status`
- `created_at`
- `updated_at`

约束：

- `mac_address` 全局唯一

### `syslog_messages`

字段建议：

- `id`
- `received_at`
- `log_time`
- `raw_message`
- `source_ip`
- `protocol`
- `parse_status`
- `retention_expire_at`

### `client_events`

字段建议：

- `id`
- `syslog_message_id`
- `event_date`
- `event_time`
- `event_type`
- `station_mac`
- `ap_mac`
- `ssid`
- `ipv4`
- `ipv6`
- `hostname`
- `os_vendor`
- `matched_employee_id`
- `match_status`

### `attendance_records`

字段建议：

- `id`
- `employee_id`
- `attendance_date`
- `first_connect_at`
- `last_disconnect_at`
- `clock_in_status`
- `clock_out_status`
- `exception_status`
- `source_mode`
- `version`
- `last_calculated_at`

约束：

- 唯一键：`(employee_id, attendance_date)`

### `attendance_reports`

字段建议：

- `id`
- `attendance_record_id`
- `report_type`
- `idempotency_key`
- `payload_json`
- `target_url`
- `report_status`
- `response_code`
- `response_body`
- `reported_at`
- `retry_count`

约束：

- `idempotency_key` 唯一

### `system_settings`

字段建议：

- `id`
- `setting_key`
- `setting_value`
- `updated_at`

首版至少包含：

- `day_end_time`
- `syslog_retention_days`
- `report_target_url`
- `report_timeout_seconds`
- `report_retry_limit`

## Idempotency

外部上报必须具备幂等保护。

建议幂等键：

- 上班：`employee_id + date + clock_in + version`
- 下班：`employee_id + date + clock_out + version`

设计收益：

- 自动重试不会重复上报
- 人工修正后版本递增，可作为新的合法业务结果重新发送

## Frontend Structure

React 控制台建议包含 5 个页面：

### 1. Dashboard

- Syslog 接收状态
- 今日连接/断开事件数
- 今日已上报上班人数
- 今日待确认下班人数
- 今日异常考勤人数
- 最近上报结果

### 2. Logs / Events

- 原始 syslog 列表
- 解析事件列表
- 按日期、MAC、员工、事件类型筛选
- 查看原始消息与解析结果

### 3. Employees & Devices

- 员工列表
- 新增/编辑员工
- 维护多个设备 MAC
- 启用/停用员工或设备

### 4. Attendance Records

- 按天查看考勤记录
- 显示上班、下班、状态、异常原因
- 支持筛选待人工处理/已上报/人工修正
- 支持手工修正与重新上报

### 5. Settings

- 日终汇总时间
- 原始 syslog 保留天数
- 外部 API 地址
- 请求超时、重试次数
- 最近一次日终任务执行状态

## Testing Strategy

首版至少包含 4 类测试：

### 1. Parser Unit Tests

验证：

- AP 日志样例能够正确提取 MAC、事件类型、时间和附属字段

### 2. Attendance Processor Unit Tests

验证：

- 第一次连接触发上班
- 多设备取最早连接
- 多次断开取最晚断开
- 无断开进入异常
- 人工修正后状态变化正确

### 3. Reporter Unit Tests

验证：

- `HTTP JSON` 请求体格式正确
- 幂等键生成正确
- 重试场景不会重复写入重复上报结果

### 4. Integration Tests

验证闭环：

- 写入一条 syslog
- 完成解析
- 匹配员工
- 聚合考勤
- 生成上报

## Delivery Scope For Init

本次 `/init` 交付目标：

- 初始化 Go 后端项目结构
- 初始化 React 前端项目结构
- 初始化 MySQL 表结构和种子配置
- 提供 `docker compose` 进行本地联调
- 实现最小可运行业务闭环
- 提供 README 说明本地启动、配置和测试方法

## Default Values

建议默认值：

- 时区：`Asia/Shanghai`
- 日终时间：`23:59`
- 原始日志保留：`30`
- 外部上报目标：单一地址
- 首版无登录鉴权

## Risks

- 不同 AP 厂商日志格式可能不同，首版仅覆盖当前样例格式
- `UDP 514` 在某些环境下可能需要额外权限或端口映射
- 如果原始日志量大，MySQL 保留策略与索引设计需要尽早验证
- 外部 API 未定稿，首版通过可配置 `HTTP JSON` 适配降低耦合

## Recommendation

推荐按“领域分层单体”实现：

- 用单体部署控制复杂度，符合 KISS
- 用模块分层隔离职责，符合 SRP
- 保留解析器、上报器的扩展点，符合 OCP
- 不提前引入队列和多服务，符合 YAGNI
