import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MarkdownContent } from "@/components/MarkdownContent";

interface QuestionBlockProps {
  question: string;
  choices: string[];
  multiSelect: boolean;
  questionIndex: number;
  totalQuestions: number;
  selectedChoices: string[];
  onChoiceToggle: (questionIndex: number, choice: string, multiSelect: boolean) => void;
  compact?: boolean;
  idPrefix?: string;
}

export function QuestionBlock({
  question,
  choices,
  multiSelect,
  questionIndex,
  totalQuestions,
  selectedChoices,
  onChoiceToggle,
  compact = false,
  idPrefix = "",
}: QuestionBlockProps) {
  const borderClass = totalQuestions > 1
    ? compact
      ? "pb-3 border-b last:border-b-0 last:pb-0"
      : "pb-4 border-b last:border-b-0 last:pb-0"
    : "";

  const counterClass = compact
    ? "text-xs text-muted-foreground mb-1.5"
    : "text-xs text-muted-foreground mb-2";

  const questionTextClass = compact
    ? "text-sm mb-2"
    : "text-sm mb-3";

  const legendClass = compact
    ? "text-sm font-medium mb-1.5"
    : "text-sm font-medium mb-2";

  const choiceSpacingClass = compact
    ? "space-y-1.5"
    : "space-y-2";

  return (
    <div className={borderClass}>
      {totalQuestions > 1 && (
        <div className={counterClass}>
          Question {questionIndex + 1} of {totalQuestions}
        </div>
      )}

      <MarkdownContent content={question} className={questionTextClass} />

      <fieldset>
        <legend className={legendClass}>
          Select your answer{multiSelect ? "(s)" : ""}:
        </legend>
        <div className={choiceSpacingClass}>
          {choices.map((choice, cIndex) => (
            <ChoiceItem
              key={cIndex}
              choice={choice}
              multiSelect={multiSelect}
              questionIndex={questionIndex}
              choiceIndex={cIndex}
              checked={selectedChoices.includes(choice)}
              onToggle={() => onChoiceToggle(questionIndex, choice, multiSelect)}
              idPrefix={idPrefix}
            />
          ))}
        </div>
      </fieldset>
    </div>
  );
}

interface ChoiceItemProps {
  choice: string;
  multiSelect: boolean;
  questionIndex: number;
  choiceIndex: number;
  checked: boolean;
  onToggle: () => void;
  idPrefix?: string;
}

function ChoiceItem({
  choice,
  multiSelect,
  questionIndex,
  choiceIndex,
  checked,
  onToggle,
  idPrefix = "",
}: ChoiceItemProps) {
  const inputId = `${idPrefix}choice-${questionIndex}-${choiceIndex}`;
  const inputName = `${idPrefix}choices_${questionIndex}`;

  return (
    <div className="flex items-start gap-2">
      <input
        type={multiSelect ? "checkbox" : "radio"}
        id={inputId}
        name={inputName}
        value={choice}
        checked={checked}
        onChange={onToggle}
        className="mt-1 shrink-0"
      />
      <Tooltip>
        <TooltipTrigger asChild>
          <label
            htmlFor={inputId}
            className="text-sm cursor-pointer overflow-hidden text-ellipsis block max-w-full"
          >
            {choice}
          </label>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-xs break-words">
          {choice}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
