import type { RouteObject } from "react-router-dom";
import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "./layout/AppShell";
import { AttendancePage } from "../features/attendance/AttendancePage";
import { DashboardPage } from "../features/dashboard/DashboardPage";
import { EmployeesPage } from "../features/employees/EmployeesPage";
import { LogsPage } from "../features/logs/LogsPage";
import { SettingsPage } from "../features/settings/SettingsPage";

export const appRoutes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "logs",
        element: <LogsPage />,
      },
      {
        path: "employees",
        element: <EmployeesPage />,
      },
      {
        path: "attendance",
        element: <AttendancePage />,
      },
      {
        path: "settings",
        element: <SettingsPage />,
      },
    ],
  },
];

export const router = createBrowserRouter(appRoutes);
