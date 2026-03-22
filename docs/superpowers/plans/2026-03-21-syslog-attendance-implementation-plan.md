# Syslog Attendance System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个可运行的 AP syslog 考勤处理系统，完成日志接收、事件解析、员工匹配、考勤汇总、外部 API 上报、人工修正和前端控制台。

**Architecture:** 采用“领域分层单体”方案，使用单个 Go 服务承载 UDP syslog 接入、业务处理、定时调度和管理 API，内部按 ingest、parser、processor、reporter、repository、scheduler 模块分层。React 前端负责控制台，MySQL 负责原始日志、标准化事件、员工资料、考勤记录和上报历史存储。

**Tech Stack:** Go 1.24+, Gin, GORM, MySQL 8, React 19, Vite, TypeScript, TanStack Query, React Router, Docker Compose, Vitest, Go test

---

## Planned File Structure

### Backend

- `backend/go.mod`
  - Go 模块定义
- `backend/cmd/server/main.go`
  - 应用启动入口，装配 HTTP 服务、UDP listener、scheduler
- `backend/internal/config/config.go`
  - 环境变量与配置加载
- `backend/internal/bootstrap/app.go`
  - 组装数据库、仓储、服务和路由
- `backend/internal/domain/attendance.go`
  - 考勤领域模型与状态常量
- `backend/internal/domain/event.go`
  - 标准化客户端事件模型
- `backend/internal/domain/report.go`
  - 上报模型与幂等键定义
- `backend/internal/repository/*.go`
  - GORM 仓储实现
- `backend/internal/parser/ap_syslog_parser.go`
  - 当前 AP 日志格式解析器
- `backend/internal/ingest/udp_listener.go`
  - UDP syslog 监听器
- `backend/internal/service/attendance_processor.go`
  - 连接/断开事件驱动的考勤聚合逻辑
- `backend/internal/service/report_service.go`
  - 外部 API 上报与幂等落库
- `backend/internal/service/day_end_service.go`
  - 日终确认与异常标记
- `backend/internal/service/settings_service.go`
  - 系统配置读写
- `backend/internal/http/router.go`
  - Gin 路由注册
- `backend/internal/http/handlers/*.go`
  - 员工、日志、考勤、配置 API
- `backend/internal/scheduler/cron.go`
  - 定时汇总与日志清理任务
- `backend/internal/db/migrations/*.sql`
  - MySQL 初始化表结构与默认数据
- `backend/tests/integration/*.go`
  - 后端集成测试

### Frontend

- `frontend/package.json`
  - 前端依赖与脚本
- `frontend/src/main.tsx`
  - React 入口
- `frontend/src/app/router.tsx`
  - 页面路由
- `frontend/src/app/layout/AppShell.tsx`
  - 控制台整体布局
- `frontend/src/lib/api.ts`
  - API 请求封装
- `frontend/src/features/dashboard/DashboardPage.tsx`
  - 总览页
- `frontend/src/features/logs/LogsPage.tsx`
  - 原始日志与解析事件页
- `frontend/src/features/employees/EmployeesPage.tsx`
  - 员工与设备管理页
- `frontend/src/features/attendance/AttendancePage.tsx`
  - 考勤记录、异常处理、人工修正页
- `frontend/src/features/settings/SettingsPage.tsx`
  - 系统配置页
- `frontend/src/components/*.tsx`
  - 复用表格、表单、状态标签组件
- `frontend/src/test/*.test.tsx`
  - 前端关键组件测试

### Root

- `docker-compose.yml`
  - MySQL、backend、frontend 联调编排
- `.env.example`
  - 默认环境变量样例
- `README.md`
  - 启动、配置、测试说明

## Task 1: Bootstrap Repository Skeleton

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `docker-compose.yml`
- Create: `.env.example`
- Create: `README.md`

- [ ] **Step 1: Write the failing smoke checks**

```bash
test -f backend/go.mod
test -f frontend/package.json
test -f docker-compose.yml
```

- [ ] **Step 2: Run check to verify it fails**

Run: `test -f backend/go.mod && test -f frontend/package.json && test -f docker-compose.yml`
Expected: non-zero exit code because files do not exist yet

- [ ] **Step 3: Write minimal project skeleton**

```go
// backend/cmd/server/main.go
package main

func main() {}
```

```json
// frontend/package.json
{
  "name": "syslog-attendance-console",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "test": "vitest run"
  }
}
```

- [ ] **Step 4: Run check to verify files exist**

Run: `test -f backend/go.mod && test -f frontend/package.json && test -f docker-compose.yml`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add backend frontend docker-compose.yml .env.example README.md
git commit -m "chore: bootstrap syslog attendance project"
```

## Task 2: Define Database Schema And Backend App Bootstrap

**Files:**
- Create: `backend/internal/db/migrations/001_init.sql`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/bootstrap/app.go`
- Create: `backend/internal/domain/attendance.go`
- Create: `backend/internal/domain/event.go`
- Create: `backend/internal/domain/report.go`
- Test: `backend/internal/config/config_test.go`

- [ ] **Step 1: Write the failing config test**

```go
func TestLoadConfigDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv(func(string) string { return "" })
	if cfg.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected default timezone Asia/Shanghai, got %s", cfg.Timezone)
	}
	if cfg.SyslogRetentionDays != 30 {
		t.Fatalf("expected retention 30, got %d", cfg.SyslogRetentionDays)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/config -run TestLoadConfigDefaults -v`
Expected: FAIL because loader does not exist

- [ ] **Step 3: Write minimal implementation**

```go
type Config struct {
	Timezone            string
	SyslogRetentionDays int
}

func LoadConfigFromEnv(getenv func(string) string) Config {
	return Config{
		Timezone:            "Asia/Shanghai",
		SyslogRetentionDays: 30,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/config -run TestLoadConfigDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal backend/go.mod
git commit -m "feat: add backend config and schema bootstrap"
```

## Task 3: Build Parser For AP Syslog Messages

**Files:**
- Create: `backend/internal/parser/ap_syslog_parser.go`
- Test: `backend/internal/parser/ap_syslog_parser_test.go`

- [ ] **Step 1: Write the failing parser test**

```go
func TestParseConnectEvent(t *testing.T) {
	raw := "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]"
	event, err := ParseAPSyslog(raw, time.Date(2026, 3, 21, 0, 33, 38, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != "connect" || event.StationMAC != "94:89:78:55:9a:f3" {
		t.Fatalf("unexpected parse result: %#v", event)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/parser -run TestParseConnectEvent -v`
Expected: FAIL because parser does not exist

- [ ] **Step 3: Write minimal parser**

```go
var stationPattern = regexp.MustCompile(`Station\[([^\]]+)\]`)

func ParseAPSyslog(raw string, receivedAt time.Time) (domain.ClientEvent, error) {
	match := stationPattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		return domain.ClientEvent{}, errors.New("station mac not found")
	}
	eventType := "disconnect"
	if strings.Contains(raw, " connect ") {
		eventType = "connect"
	}
	return domain.ClientEvent{
		EventType:  eventType,
		StationMAC: strings.ToLower(match[1]),
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/parser -run TestParseConnectEvent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/parser
git commit -m "feat: add ap syslog parser"
```

## Task 4: Implement Attendance Processor And Idempotent Reporting

**Files:**
- Create: `backend/internal/service/attendance_processor.go`
- Create: `backend/internal/service/report_service.go`
- Create: `backend/internal/repository/attendance_repository.go`
- Create: `backend/internal/repository/report_repository.go`
- Test: `backend/internal/service/attendance_processor_test.go`
- Test: `backend/internal/service/report_service_test.go`

- [ ] **Step 1: Write the failing attendance processor test**

```go
func TestFirstConnectCreatesClockIn(t *testing.T) {
	event := domain.ClientEvent{
		EventType:  "connect",
		StationMAC: "94:89:78:55:9a:f3",
		EventTime:  time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600)),
	}
	record := processor.ApplyEvent(existingRecord, matchedEmployee, event)
	if record.FirstConnectAt == nil {
		t.Fatal("expected clock-in time to be set")
	}
	if !record.ClockInNeedsReport {
		t.Fatal("expected immediate clock-in report")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/service -run TestFirstConnectCreatesClockIn -v`
Expected: FAIL because processor does not exist

- [ ] **Step 3: Write minimal implementation**

```go
func (p *AttendanceProcessor) ApplyEvent(record domain.AttendanceRecord, employee domain.Employee, event domain.ClientEvent) domain.AttendanceRecord {
	if event.EventType == "connect" && record.FirstConnectAt == nil {
		record.FirstConnectAt = &event.EventTime
		record.ClockInStatus = domain.ClockStatusPending
		record.ClockInNeedsReport = true
	}
	if event.EventType == "disconnect" {
		record.LastDisconnectAt = &event.EventTime
	}
	return record
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/service -run 'TestFirstConnectCreatesClockIn|TestBuildIdempotencyKey' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service backend/internal/repository
git commit -m "feat: add attendance processor and report service"
```

## Task 5: Add UDP Ingest, Persistence, And Day-End Scheduler

**Files:**
- Create: `backend/internal/ingest/udp_listener.go`
- Create: `backend/internal/scheduler/cron.go`
- Create: `backend/internal/service/day_end_service.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/service/day_end_service_test.go`

- [ ] **Step 1: Write the failing day-end test**

```go
func TestFinalizeMarksMissingDisconnect(t *testing.T) {
	record := domain.AttendanceRecord{ClockOutStatus: domain.ClockStatusPending}
	result := service.FinalizeForDay(record, time.Date(2026, 3, 21, 23, 59, 0, 0, time.FixedZone("CST", 8*3600)))
	if result.ExceptionStatus != domain.ExceptionMissingDisconnect {
		t.Fatalf("expected missing disconnect, got %s", result.ExceptionStatus)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/service -run TestFinalizeMarksMissingDisconnect -v`
Expected: FAIL because day-end service does not exist

- [ ] **Step 3: Write minimal implementation**

```go
func (s *DayEndService) FinalizeForDay(record domain.AttendanceRecord, now time.Time) domain.AttendanceRecord {
	if record.LastDisconnectAt == nil {
		record.ExceptionStatus = domain.ExceptionMissingDisconnect
		record.ClockOutStatus = domain.ClockStatusMissing
	}
	return record
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/service -run TestFinalizeMarksMissingDisconnect -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/ingest backend/internal/scheduler backend/internal/service
git commit -m "feat: add syslog ingest and day-end scheduler"
```

## Task 6: Expose Admin HTTP API

**Files:**
- Create: `backend/internal/http/router.go`
- Create: `backend/internal/http/handlers/employees_handler.go`
- Create: `backend/internal/http/handlers/logs_handler.go`
- Create: `backend/internal/http/handlers/attendance_handler.go`
- Create: `backend/internal/http/handlers/settings_handler.go`
- Test: `backend/internal/http/handlers/attendance_handler_test.go`

- [ ] **Step 1: Write the failing API test**

```go
func TestListAttendanceReturnsOK(t *testing.T) {
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/attendance", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/http/handlers -run TestListAttendanceReturnsOK -v`
Expected: FAIL because router/handler do not exist

- [ ] **Step 3: Write minimal implementation**

```go
func (h *AttendanceHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/http/handlers -run TestListAttendanceReturnsOK -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http
git commit -m "feat: add admin http api"
```

## Task 7: Build React Console Shell And Core Pages

**Files:**
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/app/router.tsx`
- Create: `frontend/src/app/layout/AppShell.tsx`
- Create: `frontend/src/features/dashboard/DashboardPage.tsx`
- Create: `frontend/src/features/logs/LogsPage.tsx`
- Create: `frontend/src/features/employees/EmployeesPage.tsx`
- Create: `frontend/src/features/attendance/AttendancePage.tsx`
- Create: `frontend/src/features/settings/SettingsPage.tsx`
- Test: `frontend/src/test/router.test.tsx`

- [ ] **Step 1: Write the failing router test**

```tsx
it("renders dashboard nav item", async () => {
  render(<AppShell />);
  expect(screen.getByText("Dashboard")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --runInBand src/test/router.test.tsx`
Expected: FAIL because app shell does not exist

- [ ] **Step 3: Write minimal implementation**

```tsx
export function AppShell() {
  return (
    <nav>
      <a href="/">Dashboard</a>
      <a href="/logs">日志</a>
      <a href="/employees">员工</a>
      <a href="/attendance">考勤</a>
      <a href="/settings">配置</a>
    </nav>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --runInBand src/test/router.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src frontend/package.json frontend/vite.config.ts frontend/tsconfig.json
git commit -m "feat: add frontend console shell"
```

## Task 8: Wire Frontend Data Flows For CRUD And Manual Correction

**Files:**
- Create: `frontend/src/lib/api.ts`
- Create: `frontend/src/features/employees/components/EmployeeForm.tsx`
- Create: `frontend/src/features/attendance/components/AttendanceTable.tsx`
- Create: `frontend/src/features/settings/components/SettingsForm.tsx`
- Test: `frontend/src/test/attendance-page.test.tsx`

- [ ] **Step 1: Write the failing attendance page test**

```tsx
it("shows manual correction action for exception rows", async () => {
  render(<AttendancePage />);
  expect(await screen.findByText("人工修正")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --runInBand src/test/attendance-page.test.tsx`
Expected: FAIL because attendance page is not wired

- [ ] **Step 3: Write minimal implementation**

```tsx
export function AttendanceTable() {
  return (
    <table>
      <tbody>
        <tr>
          <td>异常</td>
          <td><button>人工修正</button></td>
        </tr>
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --runInBand src/test/attendance-page.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: add frontend data workflows"
```

## Task 9: Add End-To-End Local Integration And Documentation

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Create: `backend/tests/integration/syslog_flow_test.go`
- Create: `scripts/send-sample-syslog.sh`

- [ ] **Step 1: Write the failing integration test**

```go
func TestSyslogFlow(t *testing.T) {
	t.Skip("enable after docker compose services are available")
}
```

- [ ] **Step 2: Run test to verify scaffold exists**

Run: `cd backend && go test ./tests/integration -run TestSyslogFlow -v`
Expected: PASS with skip before real implementation, proving the test target exists

- [ ] **Step 3: Write minimal integration tooling**

```bash
#!/usr/bin/env bash
echo "Mar 21 00:33:38 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[WesleyHomeEquipment] osvendor[Unknown] hostname[Wesley17PM]" | nc -u 127.0.0.1 5514
```

- [ ] **Step 4: Run verification commands**

Run:

```bash
docker compose up -d --build
cd backend && go test ./...
cd ../frontend && npm test
```

Expected:

- services start successfully
- backend tests pass
- frontend tests pass

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml README.md backend/tests/integration scripts/send-sample-syslog.sh
git commit -m "docs: add local integration workflow"
```

## Implementation Notes

- 原始 syslog 保留天数默认先定为 `30`
- 日终时间默认先定为 `23:59`
- `docker-compose.yml` 中建议把宿主机 `5514/udp` 映射到容器 `514/udp`，避免本地开发需要 root 权限
- 前端保留“重新上报”和“人工修正”两个显式动作，不做隐式覆盖
- 手工修正后应递增 `attendance_records.version`
- 后端所有时间入库统一用带时区的 `Asia/Shanghai` 语义处理

## Manual Review Checklist

- 模块职责是否单一，是否避免把解析、聚合、上报揉在同一文件
- 数据表是否足以支撑幂等、人工修正、异常处理和原始日志保留
- 前后端 API 是否覆盖 Dashboard、日志、员工、考勤、配置五个页面
- TDD 步骤是否足够细，是否每个任务都有“先写失败测试”
- README 是否覆盖本地启动、配置、样例日志发送和验证命令
