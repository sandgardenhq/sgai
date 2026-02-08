import {
  createContext,
  useContext,
  useReducer,
  type ReactNode,
} from "react";

export interface AppState {
  selectedWorkspace: string | null;
  ui: {
    panelCollapsed: boolean;
    activeTab: string;
  };
}

export type AppAction =
  | { type: "workspace/select"; workspace: string }
  | { type: "ui/togglePanel" }
  | { type: "ui/setTab"; tab: string };

const initialState: AppState = {
  selectedWorkspace: null,
  ui: {
    panelCollapsed: false,
    activeTab: "goal",
  },
};

function appReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case "workspace/select":
      return {
        ...state,
        selectedWorkspace: action.workspace,
      };
    case "ui/togglePanel":
      return {
        ...state,
        ui: {
          ...state.ui,
          panelCollapsed: !state.ui.panelCollapsed,
        },
      };
    case "ui/setTab":
      return {
        ...state,
        ui: {
          ...state.ui,
          activeTab: action.tab,
        },
      };
    default: {
      const _exhaustive: never = action;
      return state;
    }
  }
}

const AppStateContext = createContext<AppState | undefined>(undefined);
const AppDispatchContext = createContext<
  React.Dispatch<AppAction> | undefined
>(undefined);

interface AppStateProviderProps {
  children: ReactNode;
}

export function AppStateProvider({ children }: AppStateProviderProps) {
  const [state, dispatch] = useReducer(appReducer, initialState);

  return (
    <AppStateContext.Provider value={state}>
      <AppDispatchContext.Provider value={dispatch}>
        {children}
      </AppDispatchContext.Provider>
    </AppStateContext.Provider>
  );
}

export function useAppState(): AppState {
  const context = useContext(AppStateContext);
  if (context === undefined) {
    throw new Error("useAppState must be used within an AppStateProvider");
  }
  return context;
}

export function useAppDispatch(): React.Dispatch<AppAction> {
  const context = useContext(AppDispatchContext);
  if (context === undefined) {
    throw new Error("useAppDispatch must be used within an AppStateProvider");
  }
  return context;
}
