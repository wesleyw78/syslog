export type Employee = {
  id: string;
  name: string;
  team: string;
  badge: string;
  status: string;
};

export type EmployeeDraft = {
  name: string;
  team: string;
  badge: string;
};

export type AttendanceStatus = "normal" | "exception" | "corrected";

export type AttendanceRecord = {
  id: string;
  employeeName: string;
  badge: string;
  checkpoint: string;
  shift: string;
  timestamp: string;
  status: AttendanceStatus;
  note: string;
};

export type SettingsRecord = {
  scannerRetryThreshold: number;
  lateToleranceMinutes: number;
  archiveRetentionDays: number;
  manualCorrectionRequiresApproval: boolean;
};

const wait = <T,>(value: T, delay = 120): Promise<T> =>
  new Promise((resolve) => {
    globalThis.setTimeout(() => resolve(value), delay);
  });

let employeeSeed = 4;

let employees: Employee[] = [
  {
    id: "emp-1",
    name: "Lena Wu",
    team: "Assembly",
    badge: "A-447",
    status: "On shift",
  },
  {
    id: "emp-2",
    name: "Arjun Patel",
    team: "Security",
    badge: "S-118",
    status: "Briefing",
  },
  {
    id: "emp-3",
    name: "Mina Torres",
    team: "Maintenance",
    badge: "M-233",
    status: "Standby",
  },
];

let attendanceRecords: AttendanceRecord[] = [
  {
    id: "att-1",
    employeeName: "Lena Wu",
    badge: "A-447",
    checkpoint: "North Gate",
    shift: "06:00-14:00",
    timestamp: "06:02",
    status: "normal",
    note: "Auto-reconciled from badge scanner",
  },
  {
    id: "att-2",
    employeeName: "Arjun Patel",
    badge: "S-118",
    checkpoint: "Security Post",
    shift: "06:00-14:00",
    timestamp: "06:14",
    status: "exception",
    note: "Late punch outside supervisor tolerance",
  },
  {
    id: "att-3",
    employeeName: "Mina Torres",
    badge: "M-233",
    checkpoint: "Packing Line",
    shift: "14:00-22:00",
    timestamp: "13:58",
    status: "corrected",
    note: "Manual badge swap already approved",
  },
];

let settingsRecord: SettingsRecord = {
  scannerRetryThreshold: 3,
  lateToleranceMinutes: 10,
  archiveRetentionDays: 45,
  manualCorrectionRequiresApproval: true,
};

export async function listEmployees(): Promise<Employee[]> {
  return wait(employees.map((employee) => ({ ...employee })));
}

export async function createEmployee(draft: EmployeeDraft): Promise<Employee> {
  const employee: Employee = {
    id: `emp-${employeeSeed++}`,
    name: draft.name.trim(),
    team: draft.team.trim(),
    badge: draft.badge.trim().toUpperCase(),
    status: "Provisioning",
  };

  employees = [employee, ...employees];

  return wait({ ...employee });
}

export async function listAttendanceRecords(): Promise<AttendanceRecord[]> {
  return wait(attendanceRecords.map((record) => ({ ...record })));
}

export async function correctAttendanceRecord(
  recordId: string,
): Promise<AttendanceRecord> {
  const currentRecord = attendanceRecords.find((record) => record.id === recordId);

  if (!currentRecord) {
    throw new Error("Attendance record not found");
  }

  const correctedRecord: AttendanceRecord = {
    ...currentRecord,
    status: "corrected",
    note: "Manual correction queued for supervisor audit",
  };

  attendanceRecords = attendanceRecords.map((record) =>
    record.id === recordId ? correctedRecord : record,
  );

  return wait({ ...correctedRecord });
}

export async function getSettings(): Promise<SettingsRecord> {
  return wait({ ...settingsRecord });
}

export async function saveSettings(
  nextSettings: SettingsRecord,
): Promise<SettingsRecord> {
  settingsRecord = { ...nextSettings };
  return wait({ ...settingsRecord });
}
