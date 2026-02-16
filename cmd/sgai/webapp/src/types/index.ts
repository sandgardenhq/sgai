export interface Workspace {
  directory: string;
  dirName: string;
  lastModified: string;
  hasWorkspace: boolean;
  isRoot: boolean;
  running: boolean;
  needsInput: boolean;
  inProgress: boolean;
  pinned: boolean;
}

export interface WorkspaceGroup {
  root: Workspace;
  forks: Workspace[];
}

export interface SessionState {
  directory: string;
  dirName: string;
  running: boolean;
  interactiveAuto: boolean;
  status: string;
  task: string;
  currentAgent: string;
  currentModel: string;
  humanMessage: string;
  badgeClass: string;
  badgeText: string;
  needsInput: boolean;
  goalContent: string;
  projectMgmtContent: string;
  hasProjectMgmt: boolean;
  hasEditedGoal: boolean;
}

export interface ApiWorkspaceEntry {
  name: string;
  dir: string;
  running: boolean;
  needsInput: boolean;
  inProgress: boolean;
  pinned: boolean;
  isRoot: boolean;
  status: string;
  hasSgai: boolean;
  forks?: ApiWorkspaceEntry[];
}

export interface ApiWorkspacesResponse {
  workspaces: ApiWorkspaceEntry[];
}

export interface ApiAgentSequenceEntry {
  agent: string;
  elapsedTime: string;
  isCurrent: boolean;
}

export interface ApiWorkspaceForkSummary {
  name: string;
  dir: string;
  running: boolean;
  commitAhead: number;
}

export interface SessionCost {
  totalCost: number;
  inputTokens: number;
  outputTokens: number;
  cacheCreationInputTokens: number;
  cacheReadInputTokens: number;
}

export interface ApiWorkspaceDetailResponse {
  name: string;
  dir: string;
  running: boolean;
  needsInput: boolean;
  status: string;
  badgeClass: string;
  badgeText: string;
  isRoot: boolean;
  isFork: boolean;
  pinned: boolean;
  hasSgai: boolean;
  hasEditedGoal: boolean;
  interactiveAuto: boolean;
  continuousMode: boolean;
  currentAgent: string;
  currentModel: string;
  task: string;
  goalContent: string;
  rawGoalContent: string;
  fullGoalContent?: string;
  pmContent: string;
  hasProjectMgmt: boolean;
  svgHash: string;
  totalExecTime: string;
  latestProgress: string;
  agentSequence: ApiAgentSequenceEntry[];
  cost: SessionCost;
  forks?: ApiWorkspaceForkSummary[];
}

export interface ApiCreateWorkspaceRequest {
  name: string;
}

export interface ApiCreateWorkspaceResponse {
  name: string;
  dir: string;
}

export interface ProgressEntry {
  timestamp: string;
  agent: string;
  description: string;
}

export interface Message {
  id: number;
  fromAgent: string;
  toAgent: string;
  read: boolean;
  readAt: string;
  readBy: string;
  body: string;
}

export interface TodoItem {
  id: string;
  content: string;
  status: "pending" | "in_progress" | "completed" | "cancelled";
  priority: "high" | "medium" | "low";
}

export interface Agent {
  name: string;
  description: string;
}

export interface AgentsResponse {
  agents: Agent[];
}

export interface SkillSummary {
  name: string;
  fullPath: string;
  description: string;
}

export interface SkillCategory {
  name: string;
  skills: SkillSummary[];
}

export interface SkillsResponse {
  categories: SkillCategory[];
}

export interface Skill {
  name: string;
  fullPath: string;
  description: string;
  content: string;
  rawContent: string;
}

export interface SnippetSummary {
  name: string;
  fileName: string;
  fullPath: string;
  description: string;
  language: string;
}

export interface SnippetLanguage {
  name: string;
  snippets: SnippetSummary[];
}

export interface SnippetsResponse {
  languages: SnippetLanguage[];
}

export interface Snippet {
  name: string;
  fileName: string;
  language: string;
  description: string;
  whenToUse: string;
  content: string;
}

export interface MultiChoiceQuestion {
  question: string;
  choices: string[];
  multiSelect: boolean;
}

export interface PendingQuestion {
  agentName: string;
  questions: MultiChoiceQuestion[];
}

export interface ApiPendingQuestionResponse {
  questionId: string;
  type: "multi-choice" | "work-gate" | "free-text" | "";
  agentName: string;
  message: string;
  questions?: MultiChoiceQuestion[];
}

export interface ApiRespondRequest {
  questionId: string;
  answer: string;
  selectedChoices: string[];
}

export interface ApiRespondResponse {
  success: boolean;
  message: string;
}

export interface ApiSessionActionResponse {
  name: string;
  status: string;
  running: boolean;
  message: string;
}

export interface ApiSessionResponse {
  name: string;
  status: string;
  running: boolean;
  needsInput: boolean;
  interactiveAuto: boolean;
  badgeClass: string;
  badgeText: string;
  currentAgent: string;
  currentModel: string;
  task: string;
  humanMessage: string;
  latestProgress: string;
  totalExecTime: string;
  svgHash: string;
  agentSequence: ApiAgentSequenceEntry[];
  cost: ApiSessionCost;
  modelStatuses?: ApiModelStatusEntry[];
}

export interface ApiModelStatusEntry {
  modelId: string;
  status: string;
}

export interface ApiModelEntry {
  id: string;
  name: string;
}

export interface ApiModelsResponse {
  models: ApiModelEntry[];
  defaultModel?: string;
}

export interface ApiSessionCost {
  totalCost: number;
  totalTokens: ApiTokenUsage;
  byAgent: ApiAgentCost[];
}

export interface ApiTokenUsage {
  input: number;
  output: number;
  reasoning: number;
  cacheRead: number;
  cacheWrite: number;
}

export interface ApiStepCost {
  stepId: string;
  agent: string;
  cost: number;
  tokens: ApiTokenUsage;
  timestamp: string;
}

export interface ApiAgentCost {
  agent: string;
  cost: number;
  tokens: ApiTokenUsage;
  steps: ApiStepCost[];
}

export interface ApiMessageEntry {
  id: number;
  fromAgent: string;
  toAgent: string;
  body: string;
  subject: string;
  read: boolean;
  readAt?: string;
  createdAt?: string;
}

export interface ApiMessagesResponse {
  messages: ApiMessageEntry[];
}

export interface ApiTodoEntry {
  id: string;
  content: string;
  status: string;
  priority: string;
}

export interface ApiTodosResponse {
  projectTodos: ApiTodoEntry[];
  agentTodos: ApiTodoEntry[];
  currentAgent: string;
}

export interface ApiLogEntry {
  prefix: string;
  text: string;
}

export interface ApiLogResponse {
  lines: ApiLogEntry[];
}

export interface ApiDiffLine {
  lineNumber: number;
  text: string;
  class: string;
}

export interface ApiChangesResponse {
  description: string;
  diffLines: ApiDiffLine[];
}

export interface ApiCommitEntry {
  changeId: string;
  commitId: string;
  workspaces?: string[];
  timestamp: string;
  bookmarks?: string[];
  description: string;
  graphChar: string;
}

export interface ApiCommitsResponse {
  commits: ApiCommitEntry[];
}

export interface ApiEventEntry {
  timestamp: string;
  formattedTime: string;
  agent: string;
  description: string;
  showDateDivider: boolean;
  dateDivider: string;
}

export interface ApiAgentModelEntry {
  agent: string;
  models: string[];
}

export interface ApiEventsResponse {
  events: ApiEventEntry[];
  currentAgent: string;
  currentModel: string;
  svgHash: string;
  needsInput: boolean;
  humanMessage: string;
  goalContent: string;
  modelStatuses?: ApiModelStatusEntry[];
  agentModels?: ApiAgentModelEntry[];
}

export interface ApiForkCommit {
  changeId: string;
  commitId: string;
  timestamp: string;
  bookmarks?: string[];
  description: string;
}

export interface ApiForkEntry {
  name: string;
  dir: string;
  running: boolean;
  commitAhead: number;
  commits: ApiForkCommit[];
}

export interface ApiForksResponse {
  forks: ApiForkEntry[];
}

export interface ApiRetroSession {
  name: string;
  hasImprovements: boolean;
  goalSummary: string;
}

export interface ApiRetroDetail {
  sessionName: string;
  goalSummary: string;
  goalContent: string;
  improvements: string;
  improvementsRaw: string;
  hasImprovements: boolean;
  isAnalyzing: boolean;
  isApplying: boolean;
}

export interface ApiRetrospectivesResponse {
  sessions: ApiRetroSession[];
  selectedSession: string;
  details?: ApiRetroDetail;
}

export interface ApiComposerAgentConf {
  name: string;
  selected: boolean;
  model: string;
}

export interface ApiComposerState {
  description: string;
  completionGate: string;
  agents: ApiComposerAgentConf[];
  flow: string;
  tasks: string;
}

export interface ApiWizardState {
  currentStep: number;
  fromTemplate?: string;
  description?: string;
  techStack: string[];
  safetyAnalysis: boolean;
  completionGate?: string;
}

export interface ApiTechStackItem {
  id: string;
  name: string;
  selected: boolean;
}

export interface ApiComposeStateResponse {
  workspace: string;
  state: ApiComposerState;
  wizard: ApiWizardState;
  techStackItems: ApiTechStackItem[];
  flowError?: string;
}

export interface ApiComposeTemplateEntry {
  id: string;
  name: string;
  description: string;
  icon: string;
  agents: ApiComposerAgentConf[];
  flow: string;
}

export interface ApiComposeTemplatesResponse {
  templates: ApiComposeTemplateEntry[];
}

export interface ApiComposePreviewResponse {
  content: string;
  flowError?: string;
  etag: string;
}

export interface ApiComposeDraftRequest {
  state: ApiComposerState;
  wizard: ApiWizardState;
}

export interface ApiComposeDraftResponse {
  saved: boolean;
}

export interface ApiComposeSaveRequest {
  content: string;
}

export interface ApiComposeSaveResponse {
  saved: boolean;
  workspace: string;
}

// M6: Workspace Management Types

export interface ApiForkRequest {
  name: string;
}

export interface ApiForkResponse {
  name: string;
  dir: string;
  parent: string;
}

export interface ApiMergeRequest {
  forkDir: string;
  confirm: boolean;
}

export interface ApiMergeResponse {
  merged: boolean;
  message: string;
}

export interface ApiDeleteForkResponse {
  deleted: boolean;
  message: string;
}

export interface ApiRenameRequest {
  name: string;
}

export interface ApiRenameResponse {
  name: string;
  oldName: string;
  dir: string;
}

export interface ApiUpdateGoalRequest {
  content: string;
}

export interface ApiUpdateGoalResponse {
  updated: boolean;
  workspace: string;
}

export interface ApiUpdateDescriptionRequest {
  description: string;
}

export interface ApiUpdateDescriptionResponse {
  updated: boolean;
  description: string;
}

export interface ApiSteerRequest {
  message: string;
}

export interface ApiSteerResponse {
  success: boolean;
  message: string;
}

export interface ApiSelfDriveResponse {
  running: boolean;
  autoMode: boolean;
  message: string;
}

export interface ApiTogglePinResponse {
  pinned: boolean;
  message: string;
}

export interface ApiOpenEditorResponse {
  opened: boolean;
  editor: string;
  message: string;
}

export interface ApiOpenOpencodeResponse {
  opened: boolean;
  message: string;
}

export interface ApiAdhocRequest {
  prompt: string;
  model: string;
}

export interface ApiAdhocResponse {
  running: boolean;
  output: string;
  message: string;
}

export interface ApiRetroAnalyzeRequest {
  session: string;
}

export interface ApiRetroApplyRequest {
  session: string;
  selectedSuggestions: string[];
  notes?: Record<string, string>;
}

export interface ApiRetroActionResponse {
  running: boolean;
  session: string;
  message: string;
}

export type SSEEventType =
  | "workspace:update"
  | "session:update"
  | "messages:new"
  | "todos:update"
  | "log:append"
  | "changes:update"
  | "events:new"
  | "compose:update";

export interface SSEEvent {
  type: SSEEventType;
  data: unknown;
  timestamp: string;
}

export type ConnectionStatus = "connected" | "disconnected" | "reconnecting";

export interface SSEStoreSnapshot {
  connectionStatus: ConnectionStatus;
  lastEvent: SSEEvent | null;
  events: Record<SSEEventType, unknown>;
}
