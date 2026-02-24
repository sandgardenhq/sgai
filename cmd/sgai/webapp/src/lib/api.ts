import type {
  AgentsResponse,
  Skill,
  SkillsResponse,
  Snippet,
  SnippetsResponse,
  ApiPendingQuestionResponse,
  ApiRespondRequest,
  ApiRespondResponse,
  ApiSessionActionResponse,
  ApiWorkspacesResponse,
  ApiWorkspaceDetailResponse,
  ApiCreateWorkspaceResponse,
  ApiSessionResponse,
  ApiMessagesResponse,
  ApiTodosResponse,
  ApiLogResponse,
  ApiChangesResponse,
  ApiEventsResponse,
  ApiForksResponse,

  ApiComposeStateResponse,
  ApiComposeTemplatesResponse,
  ApiComposePreviewResponse,
  ApiComposeDraftRequest,
  ApiComposeDraftResponse,
  ApiComposeSaveResponse,
  ApiForkResponse,
  ApiRenameResponse,
  ApiUpdateGoalResponse,
  ApiAdhocResponse,

  ApiModelsResponse,
  ApiCommitsResponse,
  ApiSteerResponse,
  ApiUpdateDescriptionResponse,
  ApiTogglePinResponse,
  ApiOpenEditorResponse,
  ApiOpenOpencodeResponse,
  ApiDeleteForkResponse,
  ApiUpdateSummaryResponse,
} from "../types";

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const headers: HeadersInit = { ...options?.headers };
  if (options?.body) {
    (headers as Record<string, string>)["Content-Type"] = "application/json";
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const text = await response.text().catch(() => "Unknown error");
    throw new ApiError(response.status, text);
  }

  if (response.status === 204) {
    return null as T;
  }

  return response.json() as Promise<T>;
}

export const api = {
  workspaces: {
    list: () => fetchJSON<ApiWorkspacesResponse>("/api/v1/workspaces"),
    get: (name: string) =>
      fetchJSON<ApiWorkspaceDetailResponse>(`/api/v1/workspaces/${encodeURIComponent(name)}`),
    create: (name: string) =>
      fetchJSON<ApiCreateWorkspaceResponse>("/api/v1/workspaces", {
        method: "POST",
        body: JSON.stringify({ name }),
      }),
    start: (name: string, auto = false) =>
      fetchJSON<ApiSessionActionResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/start`,
        {
          method: "POST",
          body: JSON.stringify({ auto }),
        },
      ),
    stop: (name: string) =>
      fetchJSON<ApiSessionActionResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/stop`,
        { method: "POST" },
      ),
    reset: (name: string) =>
      fetchJSON<ApiSessionActionResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/reset`,
        { method: "POST" },
      ),
    respond: (name: string, request: ApiRespondRequest) =>
      fetchJSON<ApiRespondResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/respond`,
        {
          method: "POST",
          body: JSON.stringify(request),
        },
      ),
    pendingQuestion: (name: string) =>
      fetchJSON<ApiPendingQuestionResponse | null>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/pending-question`,
      ),
    messages: (name: string) =>
      fetchJSON<ApiMessagesResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/messages`,
      ),
    todos: (name: string) =>
      fetchJSON<ApiTodosResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/todos`,
      ),
    log: (name: string, lines?: number) =>
      fetchJSON<ApiLogResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/log${lines ? `?lines=${lines}` : ""}`,
      ),
    changes: (name: string) =>
      fetchJSON<ApiChangesResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/changes`,
      ),
    events: (name: string) =>
      fetchJSON<ApiEventsResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/events`,
      ),
    forks: (name: string) =>
      fetchJSON<ApiForksResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/forks`,
      ),
    session: (name: string) =>
      fetchJSON<ApiSessionResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/session`,
      ),

    commits: (name: string) =>
      fetchJSON<ApiCommitsResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/commits`,
      ),
    fork: (name: string, forkName: string) =>
      fetchJSON<ApiForkResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/fork`,
        { method: "POST", body: JSON.stringify({ name: forkName }) },
      ),
    rename: (name: string, newName: string) =>
      fetchJSON<ApiRenameResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/rename`,
        { method: "POST", body: JSON.stringify({ name: newName }) },
      ),
    updateGoal: (name: string, content: string) =>
      fetchJSON<ApiUpdateGoalResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/goal`,
        { method: "PUT", body: JSON.stringify({ content }) },
      ),
    adhoc: (name: string, prompt: string, model: string) =>
      fetchJSON<ApiAdhocResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/adhoc`,
        { method: "POST", body: JSON.stringify({ prompt, model }) },
      ),
    adhocStatus: (name: string) =>
      fetchJSON<ApiAdhocResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/adhoc`,
      ),
    adhocStop: (name: string) =>
      fetchJSON<ApiAdhocResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/adhoc`,
        { method: "DELETE" },
      ),

    steer: (name: string, message: string) =>
      fetchJSON<ApiSteerResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/steer`,
        { method: "POST", body: JSON.stringify({ message }) },
      ),
    updateDescription: (name: string, description: string) =>
      fetchJSON<ApiUpdateDescriptionResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/description`,
        { method: "POST", body: JSON.stringify({ description }) },
      ),
    togglePin: (name: string) =>
      fetchJSON<ApiTogglePinResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/pin`,
        { method: "POST" },
      ),
    openEditor: (name: string) =>
      fetchJSON<ApiOpenEditorResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/open-editor`,
        { method: "POST" },
      ),
    openOpencode: (name: string) =>
      fetchJSON<ApiOpenOpencodeResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/open-opencode`,
        { method: "POST" },
      ),
    openEditorGoal: (name: string) =>
      fetchJSON<ApiOpenEditorResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/open-editor/goal`,
        { method: "POST" },
      ),
    openEditorProjectManagement: (name: string) =>
      fetchJSON<ApiOpenEditorResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/open-editor/project-management`,
        { method: "POST" },
      ),
    deleteFork: (name: string, forkDir: string) =>
      fetchJSON<ApiDeleteForkResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/delete-fork`,
        { method: "POST", body: JSON.stringify({ forkDir, confirm: true }) },
      ),
    updateSummary: (name: string, summary: string) =>
      fetchJSON<ApiUpdateSummaryResponse>(
        `/api/v1/workspaces/${encodeURIComponent(name)}/summary`,
        { method: "PUT", body: JSON.stringify({ summary }) },
      ),
  },

  agents: {
    list: (workspace: string) =>
      fetchJSON<AgentsResponse>(
        `/api/v1/agents?workspace=${encodeURIComponent(workspace)}`,
      ),
  },

  skills: {
    list: (workspace: string) =>
      fetchJSON<SkillsResponse>(
        `/api/v1/skills?workspace=${encodeURIComponent(workspace)}`,
      ),
    get: (fullPath: string, workspace: string) =>
      fetchJSON<Skill>(
        `/api/v1/skills/${fullPath.split("/").map(encodeURIComponent).join("/")}?workspace=${encodeURIComponent(workspace)}`,
      ),
  },

  models: {
    list: (workspace: string) =>
      fetchJSON<ApiModelsResponse>(
        `/api/v1/models?workspace=${encodeURIComponent(workspace)}`,
      ),
  },

  snippets: {
    list: (workspace: string) =>
      fetchJSON<SnippetsResponse>(
        `/api/v1/snippets?workspace=${encodeURIComponent(workspace)}`,
      ),
    get: (lang: string, fileName: string, workspace: string) =>
      fetchJSON<Snippet>(
        `/api/v1/snippets/${encodeURIComponent(lang)}/${encodeURIComponent(fileName)}?workspace=${encodeURIComponent(workspace)}`,
      ),
  },

  compose: {
    get: (workspace: string) =>
      fetchJSON<ApiComposeStateResponse>(
        `/api/v1/compose?workspace=${encodeURIComponent(workspace)}`,
      ),
    save: (workspace: string, etag?: string) => {
      const headers: Record<string, string> = {};
      if (etag) {
        headers["If-Match"] = etag;
      }
      return fetchJSON<ApiComposeSaveResponse>(
        `/api/v1/compose?workspace=${encodeURIComponent(workspace)}`,
        {
          method: "POST",
          headers,
        },
      );
    },
    templates: () =>
      fetchJSON<ApiComposeTemplatesResponse>("/api/v1/compose/templates"),
    preview: (workspace: string) =>
      fetchJSON<ApiComposePreviewResponse>(
        `/api/v1/compose/preview?workspace=${encodeURIComponent(workspace)}`,
      ),
    saveDraft: (workspace: string, draft: ApiComposeDraftRequest) =>
      fetchJSON<ApiComposeDraftResponse>(
        `/api/v1/compose/draft?workspace=${encodeURIComponent(workspace)}`,
        {
          method: "POST",
          body: JSON.stringify(draft),
        },
      ),
  },
} as const;

export { ApiError };
