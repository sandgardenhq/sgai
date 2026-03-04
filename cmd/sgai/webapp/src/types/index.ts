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
  humanMessage: string;
  agentSequence: ApiAgentSequenceEntry[];
  cost: ApiSessionCost;
  modelStatuses?: ApiModelStatusEntry[];
  agentModels?: ApiAgentModelEntry[];
  events: ApiEventEntry[];
  messages: ApiMessageEntry[];
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

export interface ApiAgentSequenceEntry {
  agent: string;
  model: string;
  elapsedTime: string;
  isCurrent: boolean;
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

export interface ApiChangesData {
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

export interface ApiAgentModelEntry {
  agent: string;
  models: string[];
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

export interface ApiDeleteMessageResponse {
  deleted: boolean;
  id: number;
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

