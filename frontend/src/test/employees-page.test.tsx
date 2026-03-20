import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { EmployeesPage } from "../features/employees/EmployeesPage";
import { resetMockData } from "../lib/api";

describe("employees page", () => {
  beforeEach(() => {
    resetMockData();
  });

  it("supports mock create update and disable flows", async () => {
    render(<EmployeesPage />);

    expect(await screen.findByText("Lena Wu")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("员工姓名"), {
      target: { value: "  Chen Li  " },
    });
    fireEvent.change(screen.getByLabelText("班组"), {
      target: { value: " Logistics " },
    });
    fireEvent.change(screen.getByLabelText("工牌号"), {
      target: { value: " l-909 " },
    });
    fireEvent.click(screen.getByRole("button", { name: "新增员工" }));

    const createdCard = await screen.findByText("Chen Li");
    const createdArticle = createdCard.closest("article");

    expect(createdArticle).not.toBeNull();
    expect(within(createdArticle as HTMLElement).getByText("L-909")).toBeInTheDocument();

    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "编辑" }),
    );

    fireEvent.change(
      within(createdArticle as HTMLElement).getByLabelText("班组"),
      { target: { value: "Control Room" } },
    );
    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "保存变更" }),
    );

    expect(
      await within(createdArticle as HTMLElement).findByText("Control Room"),
    ).toBeInTheDocument();

    fireEvent.click(
      within(createdArticle as HTMLElement).getByRole("button", { name: "停用" }),
    );

    expect(
      await within(createdArticle as HTMLElement).findByText("Disabled"),
    ).toBeInTheDocument();
    expect(
      within(createdArticle as HTMLElement).getByRole("button", { name: "已停用" }),
    ).toBeDisabled();
  });

  it("rejects whitespace-only employee fields", async () => {
    render(<EmployeesPage />);

    await screen.findByText("Lena Wu");

    fireEvent.change(screen.getByLabelText("员工姓名"), {
      target: { value: "   " },
    });
    fireEvent.change(screen.getByLabelText("班组"), {
      target: { value: "   " },
    });
    fireEvent.change(screen.getByLabelText("工牌号"), {
      target: { value: "   " },
    });
    fireEvent.click(screen.getByRole("button", { name: "新增员工" }));

    expect(await screen.findByText("员工字段不能只包含空白字符")).toBeInTheDocument();
    expect(screen.queryByText("已新增员工")).not.toBeInTheDocument();
  });
});
