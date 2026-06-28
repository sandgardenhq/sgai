import { createBrowserRouter, Navigate } from "react-router";
import { App } from "./App";
import { NotYetAvailable } from "./components/NotYetAvailable";
import {
  AdhocOutputRoute,
  AgentListRoute,
  AttachExternalRoute,
  ComposeLandingRoute,
  ComposePreviewRoute,
  ComposeTemplateRoute,
  DashboardEmptyRoute,
  DashboardWorkspaceRoute,
  EditGoalRoute,
  FullDiffRoute,
  NewWorkspaceRoute,
  ResponseMultiChoiceRoute,
  SkillDetailRoute,
  SkillListRoute,
  SnippetDetailRoute,
  SnippetListRoute,
  WizardFinishRoute,
  WizardStep1Route,
  WizardStep2Route,
  WizardStep3Route,
  WizardStep4Route,
} from "./router-elements";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      {
        index: true,
        element: <DashboardEmptyRoute />,
      },
      {
        path: "trees",
        element: <Navigate to="/" replace />,
      },
      {
        path: "workspaces/new",
        element: <NewWorkspaceRoute />,
      },
      {
        path: "workspaces/attach",
        element: <AttachExternalRoute />,
      },
      {
        path: "workspaces/:name/agents",
        element: <AgentListRoute />,
      },
      {
        path: "workspaces/:name/skills",
        element: <SkillListRoute />,
      },
      {
        path: "workspaces/:name/skills/*",
        element: <SkillDetailRoute />,
      },
      {
        path: "workspaces/:name/snippets",
        element: <SnippetListRoute />,
      },
      {
        path: "workspaces/:name/snippets/:lang/:fileName",
        element: <SnippetDetailRoute />,
      },
      {
        path: "workspaces/:name/goal/edit",
        element: <EditGoalRoute />,
      },
      {
        path: "workspaces/:name/goal",
        element: <EditGoalRoute />,
      },
      {
        path: "workspaces/:name/adhoc",
        element: <AdhocOutputRoute />,
      },
      {
        path: "workspace/:name/diff",
        element: <FullDiffRoute />,
      },
      {
        path: "workspaces/:name/*",
        element: <DashboardWorkspaceRoute />,
      },
      {
        path: "workspaces/:name",
        element: <DashboardWorkspaceRoute />,
      },
      {
        path: "compose",
        element: <ComposeLandingRoute />,
      },
      {
        path: "compose/landing",
        element: <ComposeLandingRoute />,
      },
      {
        path: "compose/template/:id",
        element: <ComposeTemplateRoute />,
      },
      {
        path: "compose/step/1",
        element: <WizardStep1Route />,
      },
      {
        path: "compose/step/2",
        element: <WizardStep2Route />,
      },
      {
        path: "compose/step/3",
        element: <WizardStep3Route />,
      },
      {
        path: "compose/step/4",
        element: <WizardStep4Route />,
      },
      {
        path: "compose/finish",
        element: <WizardFinishRoute />,
      },
      {
        path: "compose/preview",
        element: <ComposePreviewRoute />,
      },
      {
        path: "workspaces/:name/respond",
        element: <ResponseMultiChoiceRoute />,
      },
      {
        path: "*",
        element: <NotYetAvailable pageName="Page" />,
      },
    ],
  },
]);
