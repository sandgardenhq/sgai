import React from "react";
import { mock, spyOn } from "bun:test";
import type { ApiWorkspaceEntry } from "@/types";

export function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/home/user/test-project",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    isRoot: false,
    isFork: false,
    status: "stopped",
    badgeClass: "",
    badgeText: "",
    hasSgai: true,
    hasEditedGoal: false,
    interactiveAuto: false,
    continuousMode: false,
    currentAgent: "",
    currentModel: "",
    task: "",
    goalContent: "",
    rawGoalContent: "",
    pmContent: "",
    hasProjectMgmt: false,
    svgHash: "",
    totalExecTime: "0s",
    latestProgress: "",
    humanMessage: "",
    agentSequence: [],
    cost: { totalCost: 0, totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 }, byAgent: [] },
    events: [],
    messages: [],
    projectTodos: [],
    agentTodos: [],
    changes: { description: "", diffLines: [] },
    commits: [],
    log: [],
    ...overrides,
  };
}

export function mockFetchSequence(responses: unknown[]) {
  let callIndex = 0;
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
    const data = responses[callIndex] ?? responses[responses.length - 1];
    callIndex++;
    return Promise.resolve(
      new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
}

export function mockFetchJson(data: unknown, status = 200) {
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
    Promise.resolve(
      new Response(JSON.stringify(data), {
        status,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
}

export function mockFetchResolved(data: unknown) {
  return spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

export function mockMarkdownEditor() {
  return (props: { value: string; onChange: (v: string | undefined) => void; disabled?: boolean; placeholder?: string }) =>
    React.createElement("textarea", {
      "data-testid": "markdown-editor",
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => props.onChange(e.target.value),
      disabled: props.disabled,
      placeholder: props.placeholder,
    });
}

export function setupMarkdownEditorMock() {
  mock.module("@monaco-editor/react", () => ({ default: () => null }));
  mock.module("@/components/MarkdownEditor", () => ({ MarkdownEditor: mockMarkdownEditor() }));
}

export function mockForkTemplateFetch() {
  return spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    if (url.includes("fork-template")) {
      return Promise.resolve(
        new Response(JSON.stringify({ content: "" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    return Promise.resolve(new Response("Not Found", { status: 404 }));
  });
}
