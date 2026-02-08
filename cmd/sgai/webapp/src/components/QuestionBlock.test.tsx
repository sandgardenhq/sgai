import { describe, it, expect, mock } from "bun:test";
import { render, screen, fireEvent } from "@testing-library/react";
import { QuestionBlock } from "./QuestionBlock";
import { TooltipProvider } from "@/components/ui/tooltip";

function renderQuestionBlock(props: Partial<React.ComponentProps<typeof QuestionBlock>> = {}) {
  const defaultProps = {
    question: "Which option?",
    choices: ["A", "B", "C"],
    multiSelect: false,
    questionIndex: 0,
    totalQuestions: 1,
    selectedChoices: [] as string[],
    onChoiceToggle: mock(() => {}),
    ...props,
  };

  return render(
    <TooltipProvider>
      <QuestionBlock {...defaultProps} />
    </TooltipProvider>,
  );
}

describe("QuestionBlock", () => {
  it("renders question text", () => {
    renderQuestionBlock();
    expect(screen.getAllByText("Which option?").length).toBeGreaterThan(0);
  });

  it("renders all choices", () => {
    renderQuestionBlock();
    expect(screen.getAllByText("A").length).toBeGreaterThan(0);
    expect(screen.getAllByText("B").length).toBeGreaterThan(0);
    expect(screen.getAllByText("C").length).toBeGreaterThan(0);
  });

  it("renders radio buttons for single select", () => {
    const { container } = renderQuestionBlock({ multiSelect: false });
    const radios = container.querySelectorAll('input[type="radio"]');
    expect(radios.length).toBe(3);
  });

  it("renders checkboxes for multi-select", () => {
    const { container } = renderQuestionBlock({ multiSelect: true });
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    expect(checkboxes.length).toBe(3);
  });

  it("shows question counter when multiple questions", () => {
    renderQuestionBlock({ questionIndex: 0, totalQuestions: 3 });
    expect(screen.getByText("Question 1 of 3")).toBeDefined();
  });

  it("does not show counter for single question", () => {
    const { container } = renderQuestionBlock({ questionIndex: 0, totalQuestions: 1 });
    const counterElements = container.querySelectorAll(".text-xs.text-muted-foreground");
    const hasQuestionCounter = Array.from(counterElements).some(
      (el) => el.textContent?.match(/Question \d+ of \d+/),
    );
    expect(hasQuestionCounter).toBe(false);
  });

  it("calls onChoiceToggle when choice clicked", () => {
    const onChoiceToggle = mock(() => {});
    const { container } = renderQuestionBlock({ onChoiceToggle });

    const radio = container.querySelector('#choice-0-0') as HTMLInputElement;
    fireEvent.click(radio);

    expect(onChoiceToggle).toHaveBeenCalledWith(0, "A", false);
  });

  it("uses idPrefix for element IDs", () => {
    const { container } = renderQuestionBlock({ idPrefix: "modal-" });
    const radio = container.querySelector('#modal-choice-0-0');
    expect(radio).not.toBeNull();
  });

  it("marks selected choices as checked", () => {
    const { container } = renderQuestionBlock({ selectedChoices: ["B"] });
    const radioB = container.querySelector('#choice-0-1') as HTMLInputElement;
    expect(radioB.checked).toBe(true);
  });

  it("applies compact spacing when compact prop is true", () => {
    renderQuestionBlock({ compact: true, totalQuestions: 2 });
    expect(screen.getByText("Question 1 of 2")).toBeDefined();
  });
});
