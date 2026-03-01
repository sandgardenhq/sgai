import type { ApiWorkspaceEntry, ApiForkEntry } from "@/types";

export function createMockWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-workspace",
    dir: "/test",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    isRoot: false,
    isFork: false,
    status: "Stopped",
    badgeClass: "",
    badgeText: "",
    hasSgai: false,
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
    totalExecTime: "",
    latestProgress: "",
    humanMessage: "",
    agentSequence: [],
    cost: {
      totalCost: 0,
      totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 },
      byAgent: [],
    },
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

export function createMockFork(overrides: Partial<ApiForkEntry> = {}): ApiForkEntry {
  return {
    name: "test-fork",
    dir: "/test-fork",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    commitAhead: 0,
    commits: [],
    ...overrides,
  };
}
