interface NotYetAvailableProps {
  pageName?: string;
}

export function NotYetAvailable({ pageName }: NotYetAvailableProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
      <h2 className="text-xl font-semibold text-muted-foreground">
        {pageName ? `${pageName} — ` : ""}Not Yet Available
      </h2>
      <p className="text-sm text-muted-foreground">
        This page is not available yet.
      </p>
    </div>
  );
}
