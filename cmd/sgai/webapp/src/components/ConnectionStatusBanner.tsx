import { useFactoryState } from "@/lib/factory-state";

export function ConnectionStatusBanner() {
  const { fetchStatus } = useFactoryState();

  if (fetchStatus !== "error") {
    return null;
  }

  return (
    <div
      role="alert"
      aria-live="polite"
      className="fixed top-0 left-0 right-0 z-50 bg-yellow-500 text-yellow-950 text-center py-2 px-4 text-sm font-medium"
    >
      Unable to fetch state from server. Retrying&hellip;
    </div>
  );
}
