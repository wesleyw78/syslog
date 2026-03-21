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
  name: string;
  status: string;
  devices: EmployeeDevice[];
  createdAt: string;
  updatedAt: string;
};

export type EmployeeUpsertInput = {
  employeeNo: string;
  systemNo: string;
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

export type SystemSetting = {
  settingKey: string;
  settingValue: string;
};

export type LogsMessage = {
  id?: string;
  receivedAt?: string;
  logTime?: string | null;
  parseStatus?: string;
  rawMessage?: string;
  sourceIp?: string;
  protocol?: string;
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

export type ListResponse<T> = {
  items: T[];
};

function buildUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function isJsonContent(response: Response): boolean {
  const contentType = response.headers.get("content-type") ?? "";
  return contentType.includes("application/json");
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
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

export function parseListResponse<T>(response: ListResponse<T> | undefined | null): T[] {
  return Array.isArray(response?.items) ? response.items : [];
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function optionalStringValue(value: unknown): string | null | undefined {
  if (typeof value === "string") {
    return value;
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

function normalizeLogMessage(value: unknown): LogsMessage {
  const raw = (value ?? {}) as Record<string, unknown>;

  return {
    id: optionalStringValue(raw.id ?? raw.ID) ?? undefined,
    receivedAt: optionalStringValue(raw.receivedAt ?? raw.ReceivedAt) ?? undefined,
    logTime: optionalStringValue(raw.logTime ?? raw.LogTime),
    parseStatus: stringValue(raw.parseStatus ?? raw.ParseStatus),
    rawMessage: stringValue(raw.rawMessage ?? raw.RawMessage),
    sourceIp: stringValue(raw.sourceIp ?? raw.SourceIP),
    protocol: stringValue(raw.protocol ?? raw.Protocol),
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

export async function getEmployees(): Promise<Employee[]> {
  const response = await apiFetch<ListResponse<Employee>>("/employees");
  return parseListResponse(response);
}

export async function createEmployee(input: EmployeeUpsertInput): Promise<Employee> {
  const response = await apiFetch<{ employee: Employee }>("/employees", {
    method: "POST",
    body: JSON.stringify(input),
  });

  return response.employee;
}

export async function updateEmployee(
  employeeId: string,
  input: EmployeeUpsertInput,
): Promise<Employee> {
  const response = await apiFetch<{ employee: Employee }>(`/employees/${employeeId}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });

  return response.employee;
}

export async function disableEmployee(employeeId: string): Promise<Employee> {
  const response = await apiFetch<{ employee: Employee }>(`/employees/${employeeId}/disable`, {
    method: "POST",
  });

  return response.employee;
}

export async function getAttendanceRecords(): Promise<AttendanceRecord[]> {
  const response = await apiFetch<ListResponse<AttendanceRecord>>("/attendance");
  return parseListResponse(response);
}

export async function correctAttendanceRecord(
  recordId: string,
  input: AttendanceCorrectionInput,
): Promise<{ attendance: AttendanceRecord; reports: unknown[] }> {
  return apiFetch<{ attendance: AttendanceRecord; reports: unknown[] }>(
    `/attendance/${recordId}/correction`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}

export async function getSettings(): Promise<SystemSetting[]> {
  const response = await apiFetch<ListResponse<unknown>>("/settings");
  return parseListResponse(response).map(normalizeSystemSetting);
}

export async function saveSettings(items: SystemSetting[]): Promise<SystemSetting[]> {
  const response = await apiFetch<ListResponse<unknown>>("/settings", {
    method: "PUT",
    body: JSON.stringify({ items }),
  });

  return parseListResponse(response).map(normalizeSystemSetting);
}

export async function getLogs(): Promise<LogItem[]> {
  const response = await apiFetch<ListResponse<unknown>>("/logs");
  return parseListResponse(response).map(normalizeLogItem);
}
