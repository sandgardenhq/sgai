import { Suspense, lazy } from "react";
import { createBrowserRouter, Navigate } from "react-router";
import { App } from "./App";
import { NotYetAvailable } from "./components/NotYetAvailable";
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
const NewFork = lazy(() =>
  import("./pages/NewFork").then((m) => ({ default: m.NewFork })),
);
const RenameFork = lazy(() =>
  import("./pages/RenameFork").then((m) => ({ default: m.RenameFork })),
);
const EditGoal = lazy(() =>
  import("./pages/EditGoal").then((m) => ({ default: m.EditGoal })),
);
const AdhocOutput = lazy(() =>
  import("./pages/AdhocOutput").then((m) => ({ default: m.AdhocOutput })),
);
const FullDiffPage = lazy(() =>
  import("./pages/FullDiffPage").then((m) => ({ default: m.FullDiffPage })),
);

function PageSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-48" />
      <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-32 rounded-xl" />
        ))}
      </div>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="flex flex-col md:flex-row gap-0 h-[calc(100vh-4rem)]">
      <aside className="w-full md:w-[280px] border-b md:border-b-0 md:border-r p-2 space-y-2">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-full rounded" />
        ))}
      </aside>
      <main className="flex-1 pt-4 md:pt-0 md:pl-4">
        <Skeleton className="h-8 w-48 mb-4" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </main>
    </div>
  );
}

function withSuspense(Component: React.ComponentType) {
  return (
    <Suspense fallback={<PageSkeleton />}>
      <Component />
    </Suspense>
  );
}

function withDashboardSuspense(Component: React.ComponentType) {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <Component />
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      {
        index: true,
        element: withDashboardSuspense(DashboardWithEmpty),
      },
      {
        path: "trees",
        element: <Navigate to="/" replace />,
      },
      {
        path: "workspaces/new",
        element: withSuspense(NewWorkspace),
      },
      {
        path: "workspaces/:name/agents",
        element: withSuspense(AgentList),
      },
      {
        path: "workspaces/:name/skills",
        element: withSuspense(SkillList),
      },
      {
        path: "workspaces/:name/skills/*",
        element: withSuspense(SkillDetail),
      },
      {
        path: "workspaces/:name/snippets",
        element: withSuspense(SnippetList),
      },
      {
        path: "workspaces/:name/snippets/:lang/:fileName",
        element: withSuspense(SnippetDetail),
      },
      {
        path: "workspaces/:name/fork/new",
        element: withSuspense(NewFork),
      },
      {
        path: "workspaces/:name/rename",
        element: withSuspense(RenameFork),
      },
      {
        path: "workspaces/:name/goal/edit",
        element: withSuspense(EditGoal),
      },
      {
        path: "workspaces/:name/goal",
        element: withSuspense(EditGoal),
      },
      {
        path: "workspaces/:name/adhoc",
        element: withSuspense(AdhocOutput),
      },
      {
        path: "workspace/:name/diff",
        element: withSuspense(FullDiffPage),
      },

      {
        path: "workspaces/:name/*",
        element: withDashboardSuspense(DashboardWithWorkspace),
      },
      {
        path: "workspaces/:name",
        element: withDashboardSuspense(DashboardWithWorkspace),
      },
      {
        path: "compose",
        element: withSuspense(ComposeLanding),
      },
      {
        path: "compose/landing",
        element: withSuspense(ComposeLanding),
      },
      {
        path: "compose/template/:id",
        element: withSuspense(ComposeTemplate),
      },
      {
        path: "compose/step/1",
        element: withSuspense(WizardStep1),
      },
      {
        path: "compose/step/2",
        element: withSuspense(WizardStep2),
      },
      {
        path: "compose/step/3",
        element: withSuspense(WizardStep3),
      },
      {
        path: "compose/step/4",
        element: withSuspense(WizardStep4),
      },
      {
        path: "compose/finish",
        element: withSuspense(WizardFinish),
      },
      {
        path: "compose/preview",
        element: withSuspense(ComposePreviewPage),
      },
      
      {
        path: "workspaces/:name/respond",
        element: withSuspense(ResponseMultiChoice),
      },
      {
        path: "*",
        element: <NotYetAvailable pageName="Page" />,
      },
    ],
  },
]);

function DashboardWithEmpty() {
  return <Dashboard><EmptyState /></Dashboard>;
}

function DashboardWithWorkspace() {
  return <Dashboard><WorkspaceDetail /></Dashboard>;
}
