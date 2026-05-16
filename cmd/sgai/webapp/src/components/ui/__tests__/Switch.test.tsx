import { afterEach, describe, expect, it } from "bun:test";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Switch } from "../switch";

describe("Switch", () => {
  afterEach(() => {
    cleanup();
  });

  it("exposes Radix switch state attributes while toggling", async () => {
    const user = userEvent.setup();
    let checked = false;

    const { rerender } = render(
      <Switch
        aria-label="Enable option"
        checked={checked}
        onCheckedChange={(nextChecked) => {
          checked = nextChecked;
        }}
      />,
    );

    const toggle = screen.getByRole("switch", { name: "Enable option" });
    expect(toggle.getAttribute("data-state")).toBe("unchecked");

    await user.click(toggle);
    rerender(
      <Switch
        aria-label="Enable option"
        checked={checked}
        onCheckedChange={(nextChecked) => {
          checked = nextChecked;
        }}
      />,
    );

    expect(toggle.getAttribute("data-state")).toBe("checked");
  });
});
