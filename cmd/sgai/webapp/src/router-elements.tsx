import { Suspense, lazy, type ReactNode } from "react";
import { Navigate, useParams } from "react-router";
import { Skeleton } from "./components/ui/skeleton";

const Dashboard = lazy(() =>
  import("./pages/Dashboard").then((m) => ({ default: m.Dashboard })),
);
const EmptyState = lazy(() =>
  import("./pages/EmptyState").then((m) => ({ default: m.EmptyState })),
);
const WorkspaceDetail = lazy(() =>
  import("./pages/WorkspaceDetail").then((m) => ({ default: m.WorkspaceDetail })),
);
const AgentList = lazy(() =>
  import("./pages/AgentList").then((m) => ({ default: m.AgentList })),
);
const SkillList = lazy(() =>
  import("./pages/SkillList").then((m) => ({ default: m.SkillList })),
);
const SkillDetail = lazy(() =>
  import("./pages/SkillDetail").then((m) => ({ default: m.SkillDetail })),
);
const SnippetList = lazy(() =>
  import("./pages/SnippetList").then((m) => ({ default: m.SnippetList })),
);
const SnippetDetail = lazy(() =>
  import("./pages/SnippetDetail").then((m) => ({ default: m.SnippetDetail })),
);
const ResponseMultiChoice = lazy(() =>
  import("./pages/ResponseMultiChoice").then((m) => ({ default: m.ResponseMultiChoice })),
);
const ComposeLanding = lazy(() =>
  import("./pages/ComposeLanding").then((m) => ({ default: m.ComposeLanding })),
);
const ComposeTemplate = lazy(() =>
  import("./pages/ComposeTemplate").then((m) => ({ default: m.ComposeTemplateRedirect })),
);
const WizardStep1 = lazy(() =>
  import("./pages/WizardStep1").then((m) => ({ default: m.WizardStep1 })),
);
const WizardStep2 = lazy(() =>
  import("./pages/WizardStep2").then((m) => ({ default: m.WizardStep2 })),
);
const WizardStep3 = lazy(() =>
  import("./pages/WizardStep3").then((m) => ({ default: m.WizardStep3 })),
);
const WizardStep4 = lazy(() =>
  import("./pages/WizardStep4").then((m) => ({ default: m.WizardStep4 })),
);
const WizardFinish = lazy(() =>
  import("./pages/WizardFinish").then((m) => ({ default: m.WizardFinish })),
);
const ComposePreviewPage = lazy(() =>
  import("./pages/ComposePreviewPage").then((m) => ({ default: m.ComposePreviewPage })),
);
const NewWorkspace = lazy(() =>
  import("./pages/NewWorkspace").then((m) => ({ default: m.NewWorkspace })),
);
const AttachExternal = lazy(() =>
  import("./pages/AttachExternal").then((m) => ({ default: m.AttachExternal })),
);
const EditGoal = lazy(() =>
  import("./pages/EditGoal").then((m) => ({ default: m.EditGoal })),
);
const AdhocOutput = lazy(() =>
  import("./pages/AdhocOutput").then((m) => ({ default: m.AdhocOutput })),
);
const Usage = lazy(() =>
  import("./pages/Usage").then((m) => ({ default: m.Usage })),
);

function PageSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-48" />
      <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
        {["page-1", "page-2", "page-3", "page-4", "page-5", "page-6"].map((key) => (
          <Skeleton key={key} className="h-32 rounded-xl" />
        ))}
      </div>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col gap-0 md:flex-row">
      <aside className="w-full space-y-2 border-b p-2 md:w-[280px] md:border-b-0 md:border-r">
        {["dashboard-1", "dashboard-2", "dashboard-3", "dashboard-4", "dashboard-5", "dashboard-6"].map((key) => (
          <Skeleton key={key} className="h-8 w-full rounded" />
        ))}
      </aside>
      <main className="flex-1 pt-4 md:pt-0 md:pl-4">
        <Skeleton className="mb-4 h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </main>
    </div>
  );
}

function DashboardWithEmpty() {
  return <Dashboard><EmptyState /></Dashboard>;
}

function DashboardWithWorkspace() {
  return <Dashboard><WorkspaceDetail /></Dashboard>;
}

function DashboardWithUsage() {
  return <Dashboard><Usage /></Dashboard>;
}

export function DashboardEmptyRoute() {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <DashboardWithEmpty />
    </Suspense>
  );
}

export function DashboardWorkspaceRoute() {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <DashboardWithWorkspace />
    </Suspense>
  );
}

export function DashboardUsageRoute() {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <DashboardWithUsage />
    </Suspense>
  );
}

export function AgentListRoute() {
  return <PageRoute><AgentList /></PageRoute>;
}

export function SkillListRoute() {
  return <PageRoute><SkillList /></PageRoute>;
}

export function SkillDetailRoute() {
  return <PageRoute><SkillDetail /></PageRoute>;
}

export function SnippetListRoute() {
  return <PageRoute><SnippetList /></PageRoute>;
}

export function SnippetDetailRoute() {
  return <PageRoute><SnippetDetail /></PageRoute>;
}

export function ResponseMultiChoiceRoute() {
  return <PageRoute><ResponseMultiChoice /></PageRoute>;
}

export function ComposeLandingRoute() {
  return <PageRoute><ComposeLanding /></PageRoute>;
}

export function ComposeTemplateRoute() {
  return <PageRoute><ComposeTemplate /></PageRoute>;
}

export function WizardStep1Route() {
  return <PageRoute><WizardStep1 /></PageRoute>;
}

export function WizardStep2Route() {
  return <PageRoute><WizardStep2 /></PageRoute>;
}

export function WizardStep3Route() {
  return <PageRoute><WizardStep3 /></PageRoute>;
}

export function WizardStep4Route() {
  return <PageRoute><WizardStep4 /></PageRoute>;
}

export function WizardFinishRoute() {
  return <PageRoute><WizardFinish /></PageRoute>;
}

export function ComposePreviewRoute() {
  return <PageRoute><ComposePreviewPage /></PageRoute>;
}

export function NewWorkspaceRoute() {
  return <PageRoute><NewWorkspace /></PageRoute>;
}

export function AttachExternalRoute() {
  return <PageRoute><AttachExternal /></PageRoute>;
}

export function EditGoalRoute() {
  return <PageRoute><EditGoal /></PageRoute>;
}

export function AdhocOutputRoute() {
  return <PageRoute><AdhocOutput /></PageRoute>;
}

export function StaleDiffRedirectRoute() {
  const { name } = useParams<{ name: string }>();
  const workspaceName = encodeURIComponent(name ?? "");
  return <Navigate to={`/workspaces/${workspaceName}/progress`} replace />;
}

function PageRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<PageSkeleton />}>{children}</Suspense>;
}
