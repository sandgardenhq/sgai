import { lazy } from "react";
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

export const AgentList = lazy(() =>
  import("./pages/AgentList").then((m) => ({ default: m.AgentList })),
);
export const SkillList = lazy(() =>
  import("./pages/SkillList").then((m) => ({ default: m.SkillList })),
);
export const SkillDetail = lazy(() =>
  import("./pages/SkillDetail").then((m) => ({ default: m.SkillDetail })),
);
export const SnippetList = lazy(() =>
  import("./pages/SnippetList").then((m) => ({ default: m.SnippetList })),
);
export const SnippetDetail = lazy(() =>
  import("./pages/SnippetDetail").then((m) => ({ default: m.SnippetDetail })),
);
export const ResponseMultiChoice = lazy(() =>
  import("./pages/ResponseMultiChoice").then((m) => ({ default: m.ResponseMultiChoice })),
);
export const ComposeLanding = lazy(() =>
  import("./pages/ComposeLanding").then((m) => ({ default: m.ComposeLanding })),
);
export const ComposeTemplate = lazy(() =>
  import("./pages/ComposeTemplate").then((m) => ({ default: m.ComposeTemplateRedirect })),
);
export const WizardStep1 = lazy(() =>
  import("./pages/WizardStep1").then((m) => ({ default: m.WizardStep1 })),
);
export const WizardStep2 = lazy(() =>
  import("./pages/WizardStep2").then((m) => ({ default: m.WizardStep2 })),
);
export const WizardStep3 = lazy(() =>
  import("./pages/WizardStep3").then((m) => ({ default: m.WizardStep3 })),
);
export const WizardStep4 = lazy(() =>
  import("./pages/WizardStep4").then((m) => ({ default: m.WizardStep4 })),
);
export const WizardFinish = lazy(() =>
  import("./pages/WizardFinish").then((m) => ({ default: m.WizardFinish })),
);
export const ComposePreviewPage = lazy(() =>
  import("./pages/ComposePreviewPage").then((m) => ({ default: m.ComposePreviewPage })),
);
export const NewWorkspace = lazy(() =>
  import("./pages/NewWorkspace").then((m) => ({ default: m.NewWorkspace })),
);
export const AttachExternal = lazy(() =>
  import("./pages/AttachExternal").then((m) => ({ default: m.AttachExternal })),
);
export const EditGoal = lazy(() =>
  import("./pages/EditGoal").then((m) => ({ default: m.EditGoal })),
);
export const AdhocOutput = lazy(() =>
  import("./pages/AdhocOutput").then((m) => ({ default: m.AdhocOutput })),
);
export const FullDiffPage = lazy(() =>
  import("./pages/FullDiffPage").then((m) => ({ default: m.FullDiffPage })),
);

export function PageSkeleton() {
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

export function DashboardSkeleton() {
  return (
    <div className="flex flex-col md:flex-row gap-0 h-[calc(100vh-4rem)]">
      <aside className="w-full md:w-[280px] border-b md:border-b-0 md:border-r p-2 space-y-2">
        {["dashboard-1", "dashboard-2", "dashboard-3", "dashboard-4", "dashboard-5", "dashboard-6"].map((key) => (
          <Skeleton key={key} className="h-8 w-full rounded" />
        ))}
      </aside>
      <main className="flex-1 pt-4 md:pt-0 md:pl-4">
        <Skeleton className="h-8 w-48 mb-4" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </main>
    </div>
  );
}

export function DashboardWithEmpty() {
  return <Dashboard><EmptyState /></Dashboard>;
}

export function DashboardWithWorkspace() {
  return <Dashboard><WorkspaceDetail /></Dashboard>;
}
