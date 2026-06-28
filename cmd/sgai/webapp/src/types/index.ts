export interface ApiWorkspaceEntry {
  name: string;
  dir: string;
  running: boolean;
  needsInput: boolean;
  inProgress: boolean;
  pinned: boolean;
  isRoot: boolean;
  isFork: boolean;
  description: string;
  status: string;
  badgeClass: string;
  badgeText: string;
  hasSgai: boolean;
  hasEditedGoal: boolean;
  interactiveAuto: boolean;
  continuousMode: boolean;
  task: string;
  goalContent: string;
  rawGoalContent: string;
  fullGoalContent?: string;
  pmContent: string;
  hasProjectMgmt: boolean;
  totalExecTime: string;
  latestProgress: string;
  humanMessage: string;
  events: ApiEventEntry[];
  projectTodos: ApiTodoEntry[];
  agentTodos: ApiTodoEntry[];
  changes: ApiChangesData;
  commits: ApiCommitEntry[];
  forks?: ApiForkEntry[];
  log: ApiLogEntry[];
  pendingQuestion?: ApiPendingQuestionResponse;
  actions?: ApiActionEntry[];
  external?: boolean;
}

export interface ApiActionEntry {
  name: string;
  model: string;
  prompt: string;
  description?: string;
}

export interface ApiGoalResponse {
  content: string;
}

export interface ApiCreateWorkspaceResponse {
  name: string;
  dir: string;
}

export interface Agent {
  name: string;
  description: string;
}

export interface AgentsResponse {
  agents: Agent[];
}

interface SkillSummary {
  name: string;
  fullPath: string;
  description: string;
}

interface SkillCategory {
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

interface SnippetSummary {
  name: string;
  fileName: string;
  fullPath: string;
  description: string;
  language: string;
}

interface SnippetLanguage {
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

interface MultiChoiceQuestion {
  question: string;
  choices: string[];
  multiSelect: boolean;
}

export interface ApiPendingQuestionResponse {
  questionId: string;
  type: "multi-choice" | "work-gate" | "free-text" | "";
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

export interface ApiModelEntry {
  id: string;
  name: string;
}

export interface ApiModelsResponse {
  models: ApiModelEntry[];
  defaultModel?: string;
}

export interface ApiTokenUsageRow {
  agent: string;
  model: string;
  input: number;
  output: number;
  cacheRead: number;
  cacheWrite: number;
  reasoning: number;
  other: number;
  total: number;
  sessionCount: number;
}

export interface ApiTokenUsageResponse {
  rows: ApiTokenUsageRow[];
  totals: ApiTokenUsageRow;
}

export interface ApiTodoEntry {
  id: string;
  content: string;
  status: string;
  priority: string;
}

export interface ApiLogEntry {
  prefix: string;
  text: string;
}

export interface ApiDiffLine {
  lineNumber: number;
  text: string;
  class: string;
}

interface ApiChangesData {
  description: string;
  diffLines: ApiDiffLine[];
}

export interface ApiDiffResponse {
  diff: string;
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

export interface ApiEventEntry {
  timestamp: string;
  formattedTime: string;
  agent: string;
  description: string;
  showDateDivider: boolean;
  dateDivider: string;
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
  needsInput: boolean;
  inProgress: boolean;
  pinned: boolean;
  description: string;
  commitAhead: number;
  commits: ApiForkCommit[];
}

export interface ApiComposerAgentConf {
  name: string;
  selected: boolean;
}

export interface ApiComposerState {
  description: string;
  completionGate: string;
  retrospective: boolean;
  agents: ApiComposerAgentConf[];
  model: string;
  tasks: string;
}

export interface ApiWizardState {
  currentStep: number;
  fromTemplate?: string;
  description?: string;
  techStack: string[];
  safetyAnalysis: boolean;
  completionGate?: string;
  retrospective: boolean;
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
}

export interface ApiComposeTemplateEntry {
  id: string;
  name: string;
  description: string;
  icon: string;
  agents: ApiComposerAgentConf[];
}

export interface ApiComposeTemplatesResponse {
  templates: ApiComposeTemplateEntry[];
}

export interface ApiComposePreviewResponse {
  content: string;
  etag: string;
}

export interface ApiComposeDraftRequest {
  state: ApiComposerState;
  wizard: ApiWizardState;
}

export interface ApiComposeDraftResponse {
  saved: boolean;
}

export interface ApiComposeSaveResponse {
  saved: boolean;
  workspace: string;
}

export interface ApiForkResponse {
  name: string;
  dir: string;
  parent: string;
}

export interface ApiDeleteForkResponse {
  deleted: boolean;
  message: string;
}

export interface ApiDeleteWorkspaceResponse {
  deleted: boolean;
  message: string;
}

export interface ApiResetWorkspaceResponse {
  reset: boolean;
  message: string;
}

export interface ApiUpdateGoalResponse {
  updated: boolean;
  workspace: string;
}

export interface ApiSteerResponse {
  success: boolean;
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

export interface ApiAdhocResponse {
  running: boolean;
  output: string;
  message: string;
}

export interface ApiForkTemplateResponse {
  content: string;
}

export interface ApiAttachWorkspaceResponse {
  name: string;
  dir: string;
  hasGoal: boolean;
}

export interface ApiDetachWorkspaceResponse {
  detached: boolean;
  message: string;
}

export interface ApiBrowseDirectoryEntry {
  name: string;
  path: string;
}

export interface ApiBrowseDirectoriesResponse {
  entries: ApiBrowseDirectoryEntry[];
}
