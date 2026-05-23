import { Suspense } from "react";
import { createBrowserRouter, Navigate } from "react-router";
import { App } from "./App";
import { NotYetAvailable } from "./components/NotYetAvailable";
import {
  AdhocOutput,
  AgentList,
  AttachExternal,
  ComposeLanding,
  ComposePreviewPage,
  ComposeTemplate,
  DashboardSkeleton,
  DashboardWithEmpty,
  DashboardWithWorkspace,
  EditGoal,
  FullDiffPage,
  NewWorkspace,
  PageSkeleton,
  ResponseMultiChoice,
  SkillDetail,
  SkillList,
  SnippetDetail,
  SnippetList,
  WizardFinish,
  WizardStep1,
  WizardStep2,
  WizardStep3,
  WizardStep4,
} from "./router-components";

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
        path: "workspaces/attach",
        element: withSuspense(AttachExternal),
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
