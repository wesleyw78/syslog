const API_BASE_URL = "/api";

export type EmployeeDevice = {
  macAddress: string;
  deviceLabel: string;
  status: string;
};

export type Employee = {
  id: string;
  employeeNo: string;
  systemNo: string;
  feishuEmployeeId: string;
  name: string;
  status: string;
  devices: EmployeeDevice[];
  createdAt: string;
  updatedAt: string;
};

export type EmployeeUpsertInput = {
  employeeNo: string;
  systemNo: string;
  feishuEmployeeId: string;
  name: string;
  status: string;
  devices: EmployeeDevice[];
};

export type AttendanceRecord = {
  id: string;
  employeeId: string;
  attendanceDate: string;
  firstConnectAt?: string | null;
  lastDisconnectAt?: string | null;
  clockInStatus: string;
  clockOutStatus: string;
  exceptionStatus: string;
  sourceMode: string;
  version: number;
  lastCalculatedAt?: string | null;
};

export type AttendanceCorrectionInput = {
  firstConnectAt?: string | null;
  lastDisconnectAt?: string | null;
};

export type DebugSyslogInjectInput = {
  rawMessage: string;
  receivedAt: string;
};

export type DebugSyslogInjectResult = {
  accepted: boolean;
  receivedAt: string;
  parseStatus: string;
  parseError: string;
};

export type DebugAttendanceDispatchInput = {
  reportType: "clock_in" | "clock_out";
};

export type DebugAttendanceReport = {
  id: string;
  attendanceRecordId: string;
  reportType: string;
  reportStatus: string;
  notificationStatus: string;
  notificationMessageId: string;
  notificationResponseCode?: number | null;
  notificationResponseBody: string;
  notificationSentAt?: string | null;
  notificationRetryCount: number;
  responseCode?: number | null;
  responseBody: string;
  externalRecordId: string;
  deleteRecordId: string;
  reportedAt?: string | null;
};

export type DebugAttendanceDispatchResult = {
  attendance: AttendanceRecord;
  report: DebugAttendanceReport;
};

export type SystemSetting = {
  settingKey: string;
  settingValue: string;
};

export type SyslogReceiveRule = {
  id: string;
  sortOrder: number;
  name: string;
  enabled: boolean;
  eventType: "connect" | "disconnect";
  messagePattern: string;
  stationMacGroup: string;
  apMacGroup: string;
  ssidGroup: string;
  ipv4Group: string;
  ipv6Group: string;
  hostnameGroup: string;
  osVendorGroup: string;
  eventTimeGroup: string;
  eventTimeLayout: string;
  createdAt?: string;
  updatedAt?: string;
};

export type SyslogReceiveRuleInput = Omit<
  SyslogReceiveRule,
  "id" | "sortOrder" | "createdAt" | "updatedAt"
>;

export type LogsMessage = {
  id?: string;
  receivedAt?: string;
  logTime?: string | null;
  parseStatus?: string;
  rawMessage?: string;
  sourceIp?: string;
  protocol?: string;
  matchedRuleName?: string;
};

export type ClientEvent = {
  id?: string;
  eventTime?: string;
  eventType?: string;
  stationMac?: string;
  hostname?: string;
  matchStatus?: string;
};

export type LogItem = {
  message: LogsMessage;
  event?: ClientEvent;
};

export type PaginationMeta = {
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
};

export type ListResponse<T> = {
  items: T[];
};

export type PaginatedResponse<T> = {
  items: T[];
  pagination: PaginationMeta;
};

export type GetLogsInput = {
  fromDate?: string;
  page?: number;
  query?: string;
  scope?: "matched" | "all";
  toDate?: string;
};

export type SyslogRulePreviewInput = {
  rawMessage: string;
  receivedAt: string;
  rule: SyslogReceiveRuleInput;
};

export type SyslogRulePreviewResult = {
  matched: boolean;
  event?: ClientEvent;
};

function buildUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function isJsonContent(response: Response): boolean {
  const contentType = response.headers.get("content-type") ?? "";
  return contentType.includes("application/json");
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(buildUrl(path), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  if (isJsonContent(response)) {
    return (await response.json()) as T;
  }

  return (await response.text()) as T;
}

export function parseListResponse<T>(
  response: ListResponse<T> | undefined | null,
): T[] {
  return Array.isArray(response?.items) ? response.items : [];
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function stringIdValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  return "";
}

function optionalStringValue(value: unknown): string | null | undefined {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  if (value === null) {
    return null;
  }
  return undefined;
}

function normalizeSystemSetting(value: unknown): SystemSetting {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    settingKey: stringValue(raw.settingKey ?? raw.SettingKey),
    settingValue: stringValue(raw.settingValue ?? raw.SettingValue),
  };
}

function booleanValue(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "number") {
    return value !== 0;
  }
  return false;
}

function normalizeSyslogReceiveRule(value: unknown): SyslogReceiveRule {
  const raw = (value ?? {}) as Record<string, unknown>;

  const eventType = stringValue(raw.eventType ?? raw.EventType);
  return {
    id: stringIdValue(raw.id ?? raw.ID),
    sortOrder:
      typeof raw.sortOrder === "number"
        ? raw.sortOrder
        : typeof raw.SortOrder === "number"
          ? raw.SortOrder
          : 0,
    name: stringValue(raw.name ?? raw.Name),
    enabled: booleanValue(raw.enabled ?? raw.Enabled),
    eventType: eventType === "disconnect" ? "disconnect" : "connect",
    messagePattern: stringValue(raw.messagePattern ?? raw.MessagePattern),
    stationMacGroup: stringValue(raw.stationMacGroup ?? raw.StationMacGroup),
    apMacGroup: stringValue(raw.apMacGroup ?? raw.APMacGroup),
    ssidGroup: stringValue(raw.ssidGroup ?? raw.SSIDGroup),
    ipv4Group: stringValue(raw.ipv4Group ?? raw.IPv4Group),
    ipv6Group: stringValue(raw.ipv6Group ?? raw.IPv6Group),
    hostnameGroup: stringValue(raw.hostnameGroup ?? raw.HostnameGroup),
    osVendorGroup: stringValue(raw.osVendorGroup ?? raw.OSVendorGroup),
    eventTimeGroup: stringValue(raw.eventTimeGroup ?? raw.EventTimeGroup),
    eventTimeLayout: stringValue(raw.eventTimeLayout ?? raw.EventTimeLayout),
    createdAt: optionalStringValue(raw.createdAt ?? raw.CreatedAt) ?? undefined,
    updatedAt: optionalStringValue(raw.updatedAt ?? raw.UpdatedAt) ?? undefined,
  };
}

function normalizeLogMessage(value: unknown): LogsMessage {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    id: optionalStringValue(raw.id ?? raw.ID) ?? undefined,
    receivedAt:
      optionalStringValue(raw.receivedAt ?? raw.ReceivedAt) ?? undefined,
    logTime: optionalStringValue(raw.logTime ?? raw.LogTime),
    parseStatus: stringValue(raw.parseStatus ?? raw.ParseStatus),
    rawMessage: stringValue(raw.rawMessage ?? raw.RawMessage),
    sourceIp: stringValue(raw.sourceIp ?? raw.SourceIP),
    protocol: stringValue(raw.protocol ?? raw.Protocol),
    matchedRuleName: stringValue(raw.matchedRuleName ?? raw.MatchedRuleName),
  };
}

function normalizeClientEvent(value: unknown): ClientEvent | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }

  const raw = value as Record<string, unknown>;

  return {
    id: optionalStringValue(raw.id ?? raw.ID) ?? undefined,
    eventTime: optionalStringValue(raw.eventTime ?? raw.EventTime) ?? undefined,
    eventType: stringValue(raw.eventType ?? raw.EventType),
    stationMac: stringValue(raw.stationMac ?? raw.StationMac),
    hostname: stringValue(raw.hostname ?? raw.Hostname),
    matchStatus: stringValue(raw.matchStatus ?? raw.MatchStatus),
  };
}

function normalizeLogItem(value: unknown): LogItem {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    message: normalizeLogMessage(raw.message),
    event: normalizeClientEvent(raw.event),
  };
}

function normalizePaginationMeta(value: unknown): PaginationMeta {
  const raw = (value ?? {}) as Record<string, unknown>;
  const page =
    typeof raw.page === "number" && Number.isFinite(raw.page)
      ? raw.page
      : typeof raw.Page === "number" && Number.isFinite(raw.Page)
        ? raw.Page
        : 1;
  const pageSize =
    typeof raw.pageSize === "number" && Number.isFinite(raw.pageSize)
      ? raw.pageSize
      : typeof raw.PageSize === "number" && Number.isFinite(raw.PageSize)
        ? raw.PageSize
        : 10;
  const totalItems =
    typeof raw.totalItems === "number" && Number.isFinite(raw.totalItems)
      ? raw.totalItems
      : typeof raw.TotalItems === "number" && Number.isFinite(raw.TotalItems)
        ? raw.TotalItems
        : 0;
  const totalPages =
    typeof raw.totalPages === "number" && Number.isFinite(raw.totalPages)
      ? raw.totalPages
      : typeof raw.TotalPages === "number" && Number.isFinite(raw.TotalPages)
        ? raw.TotalPages
        : 0;

  return {
    page,
    pageSize,
    totalItems,
    totalPages,
  };
}

function normalizeEmployeeDevice(value: unknown): EmployeeDevice {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    macAddress: stringValue(raw.macAddress ?? raw.MacAddress),
    deviceLabel: stringValue(raw.deviceLabel ?? raw.DeviceLabel),
    status: stringValue(raw.status ?? raw.Status),
  };
}

function normalizeEmployee(value: unknown): Employee {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    id: stringIdValue(raw.id ?? raw.ID),
    employeeNo: stringValue(raw.employeeNo ?? raw.EmployeeNo),
    systemNo: stringValue(raw.systemNo ?? raw.SystemNo),
    feishuEmployeeId: stringValue(
      raw.feishuEmployeeId ?? raw.FeishuEmployeeID,
    ),
    name: stringValue(raw.name ?? raw.Name),
    status: stringValue(raw.status ?? raw.Status),
    devices: Array.isArray(raw.devices ?? raw.Devices)
      ? ((raw.devices ?? raw.Devices) as unknown[]).map(normalizeEmployeeDevice)
      : [],
    createdAt: stringValue(raw.createdAt ?? raw.CreatedAt),
    updatedAt: stringValue(raw.updatedAt ?? raw.UpdatedAt),
  };
}

function normalizeAttendanceRecord(value: unknown): AttendanceRecord {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    id: stringIdValue(raw.id ?? raw.ID),
    employeeId: stringIdValue(raw.employeeId ?? raw.EmployeeID),
    attendanceDate: stringValue(raw.attendanceDate ?? raw.AttendanceDate),
    firstConnectAt: optionalStringValue(
      raw.firstConnectAt ?? raw.FirstConnectAt,
    ),
    lastDisconnectAt: optionalStringValue(
      raw.lastDisconnectAt ?? raw.LastDisconnectAt,
    ),
    clockInStatus: stringValue(raw.clockInStatus ?? raw.ClockInStatus),
    clockOutStatus: stringValue(raw.clockOutStatus ?? raw.ClockOutStatus),
    exceptionStatus: stringValue(raw.exceptionStatus ?? raw.ExceptionStatus),
    sourceMode: stringValue(raw.sourceMode ?? raw.SourceMode),
    version: typeof raw.version === "number" ? raw.version : 0,
    lastCalculatedAt: optionalStringValue(
      raw.lastCalculatedAt ?? raw.LastCalculatedAt,
    ),
  };
}

function normalizeDebugSyslogResult(value: unknown): DebugSyslogInjectResult {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    accepted: Boolean(raw.accepted ?? raw.Accepted),
    receivedAt: stringValue(raw.receivedAt ?? raw.ReceivedAt),
    parseStatus: stringValue(raw.parseStatus ?? raw.ParseStatus),
    parseError: stringValue(raw.parseError ?? raw.ParseError),
  };
}

function normalizeDebugAttendanceReport(value: unknown): DebugAttendanceReport {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    id: stringIdValue(raw.id ?? raw.ID),
    attendanceRecordId: stringIdValue(
      raw.attendanceRecordId ?? raw.AttendanceRecordID,
    ),
    reportType: stringValue(raw.reportType ?? raw.ReportType),
    reportStatus: stringValue(raw.reportStatus ?? raw.ReportStatus),
    notificationStatus: stringValue(
      raw.notificationStatus ?? raw.NotificationStatus,
    ),
    notificationMessageId: stringValue(
      raw.notificationMessageId ?? raw.NotificationMessageID,
    ),
    notificationResponseCode:
      typeof raw.notificationResponseCode === "number"
        ? raw.notificationResponseCode
        : typeof raw.NotificationResponseCode === "number"
          ? raw.NotificationResponseCode
          : null,
    notificationResponseBody: stringValue(
      raw.notificationResponseBody ?? raw.NotificationResponseBody,
    ),
    notificationSentAt: optionalStringValue(
      raw.notificationSentAt ?? raw.NotificationSentAt,
    ),
    notificationRetryCount:
      typeof raw.notificationRetryCount === "number"
        ? raw.notificationRetryCount
        : typeof raw.NotificationRetryCount === "number"
          ? raw.NotificationRetryCount
          : 0,
    responseCode:
      typeof raw.responseCode === "number"
        ? raw.responseCode
        : typeof raw.ResponseCode === "number"
          ? raw.ResponseCode
          : null,
    responseBody: stringValue(raw.responseBody ?? raw.ResponseBody),
    externalRecordId: stringValue(raw.externalRecordId ?? raw.ExternalRecordID),
    deleteRecordId: stringValue(raw.deleteRecordId ?? raw.DeleteRecordID),
    reportedAt: optionalStringValue(raw.reportedAt ?? raw.ReportedAt),
  };
}

export async function getEmployees(): Promise<Employee[]> {
  const response = await apiFetch<ListResponse<unknown>>("/employees");
  return parseListResponse(response).map(normalizeEmployee);
}

export async function createEmployee(
  input: EmployeeUpsertInput,
): Promise<Employee> {
  const response = await apiFetch<{ employee: unknown }>("/employees", {
    method: "POST",
    body: JSON.stringify(input),
  });

  return normalizeEmployee(response.employee);
}

export async function updateEmployee(
  employeeId: string,
  input: EmployeeUpsertInput,
): Promise<Employee> {
  const response = await apiFetch<{ employee: unknown }>(
    `/employees/${employeeId}`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
  );

  return normalizeEmployee(response.employee);
}

export async function disableEmployee(employeeId: string): Promise<Employee> {
  const response = await apiFetch<{ employee: unknown }>(
    `/employees/${employeeId}/disable`,
    {
      method: "POST",
    },
  );

  return normalizeEmployee(response.employee);
}

export async function getAttendanceRecords(): Promise<AttendanceRecord[]> {
  const response = await apiFetch<ListResponse<unknown>>("/attendance");
  return parseListResponse(response).map(normalizeAttendanceRecord);
}

export async function correctAttendanceRecord(
  recordId: string,
  input: AttendanceCorrectionInput,
): Promise<{ attendance: AttendanceRecord; reports: unknown[] }> {
  const response = await apiFetch<{ attendance: unknown; reports: unknown[] }>(
    `/attendance/${recordId}/correction`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );

  return {
    attendance: normalizeAttendanceRecord(response.attendance),
    reports: Array.isArray(response.reports) ? response.reports : [],
  };
}

export async function getSettings(): Promise<SystemSetting[]> {
  const response = await apiFetch<ListResponse<unknown>>("/settings");
  return parseListResponse(response).map(normalizeSystemSetting);
}

export async function saveSettings(
  items: SystemSetting[],
): Promise<SystemSetting[]> {
  const response = await apiFetch<ListResponse<unknown>>("/settings", {
    method: "PUT",
    body: JSON.stringify({ items }),
  });

  return parseListResponse(response).map(normalizeSystemSetting);
}

export async function getSyslogRules(): Promise<SyslogReceiveRule[]> {
  const response = await apiFetch<ListResponse<unknown>>("/syslog-rules");
  return parseListResponse(response).map(normalizeSyslogReceiveRule);
}

export async function createSyslogRule(
  input: SyslogReceiveRuleInput,
): Promise<SyslogReceiveRule> {
  const response = await apiFetch<unknown>("/syslog-rules", {
    method: "POST",
    body: JSON.stringify(input),
  });

  return normalizeSyslogReceiveRule(response);
}

export async function updateSyslogRule(
  ruleId: string,
  input: SyslogReceiveRuleInput,
): Promise<SyslogReceiveRule> {
  const response = await apiFetch<unknown>(`/syslog-rules/${ruleId}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });

  return normalizeSyslogReceiveRule(response);
}

export async function deleteSyslogRule(ruleId: string): Promise<void> {
  await apiFetch<void>(`/syslog-rules/${ruleId}`, {
    method: "DELETE",
  });
}

export async function moveSyslogRule(
  ruleId: string,
  direction: "up" | "down",
): Promise<SyslogReceiveRule> {
  const response = await apiFetch<unknown>(`/syslog-rules/${ruleId}/move`, {
    method: "POST",
    body: JSON.stringify({ direction }),
  });

  return normalizeSyslogReceiveRule(response);
}

export async function previewSyslogRule(
  input: SyslogRulePreviewInput,
): Promise<SyslogRulePreviewResult> {
  const response = await apiFetch<unknown>("/syslog-rules/preview", {
    method: "POST",
    body: JSON.stringify(input),
  });
  const raw = (response ?? {}) as Record<string, unknown>;

  return {
    matched: Boolean(raw.matched ?? raw.Matched),
    event: normalizeClientEvent(raw.event ?? raw.Event),
  };
}

function buildLogsQuery(input: GetLogsInput = {}): string {
  const params = new URLSearchParams();
  const page =
    typeof input.page === "number" &&
    Number.isFinite(input.page) &&
    input.page > 0
      ? Math.floor(input.page)
      : 1;

  params.set("page", String(page));

  if (input.query && input.query.trim() !== "") {
    params.set("query", input.query.trim());
  }

  if (input.fromDate && input.fromDate.trim() !== "") {
    params.set("fromDate", input.fromDate.trim());
  }

  if (input.toDate && input.toDate.trim() !== "") {
    params.set("toDate", input.toDate.trim());
  }

  if (input.scope === "all") {
    params.set("scope", "all");
  }

  return `/logs?${params.toString()}`;
}

export async function getLogs(
  input: GetLogsInput = {},
): Promise<PaginatedResponse<LogItem>> {
  const response = await apiFetch<PaginatedResponse<unknown>>(
    buildLogsQuery(input),
  );

  return {
    items: parseListResponse(response).map(normalizeLogItem),
    pagination: normalizePaginationMeta(response?.pagination),
  };
}

export async function injectDebugSyslog(
  input: DebugSyslogInjectInput,
): Promise<DebugSyslogInjectResult> {
  const response = await apiFetch<unknown>("/debug/syslog", {
    method: "POST",
    body: JSON.stringify(input),
  });

  return normalizeDebugSyslogResult(response);
}

export async function dispatchAttendanceReport(
  recordId: string,
  input: DebugAttendanceDispatchInput,
): Promise<DebugAttendanceDispatchResult> {
  const response = await apiFetch<{ attendance: unknown; report: unknown }>(
    `/debug/attendance/${recordId}/dispatch`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );

  return {
    attendance: normalizeAttendanceRecord(response.attendance),
    report: normalizeDebugAttendanceReport(response.report),
  };
}
