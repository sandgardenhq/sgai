import { useCallback, useEffect, useMemo, useReducer, type ComponentType } from "react";
import { useSearchParams } from "react-router";
import { AlertCircle, Coins, Database, RefreshCw, Sigma } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectOption } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { ApiUsageDailyPoint, ApiUsageResponse, ApiUsageRow, ApiUsageTokenUsage } from "@/types";

const currencyFormatter = new Intl.NumberFormat(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 4 });
const numberFormatter = new Intl.NumberFormat();

function defaultFromDate(): string {
  const date = new Date();
  date.setDate(date.getDate() - 30);
  return date.toISOString().slice(0, 10);
}

function todayDate(): string {
  return new Date().toISOString().slice(0, 10);
}

function formatCurrency(value: number): string {
  return currencyFormatter.format(value);
}

function formatNumber(value: number): string {
  return numberFormatter.format(value);
}

function tokenTotal(tokens: ApiUsageTokenUsage): number {
  return tokens.input + tokens.output + tokens.reasoning + tokens.cacheRead + tokens.cacheWrite;
}

function compactTokens(tokens: ApiUsageTokenUsage): string {
  return formatNumber(tokenTotal(tokens));
}

interface TruncatedNameProps {
  value: string;
  path?: string;
  className?: string;
}

function TruncatedName({ value, path, className }: TruncatedNameProps) {
  const label = value || "—";
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn("block max-w-[16rem] overflow-hidden text-ellipsis whitespace-nowrap", className)}>
          {label}
        </span>
      </TooltipTrigger>
      <TooltipContent>
        <div className="max-w-sm space-y-1">
          <div className="font-medium">{label}</div>
          {path ? <div className="text-xs text-muted-foreground break-all">{path}</div> : null}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function UsageSkeleton() {
  return (
    <div className="space-y-4" aria-label="Loading usage">
      <Skeleton className="h-10 w-72" />
      <div className="grid gap-3 md:grid-cols-3">
        <Skeleton className="h-32 rounded-xl" />
        <Skeleton className="h-32 rounded-xl" />
        <Skeleton className="h-32 rounded-xl" />
      </div>
      <Skeleton className="h-72 rounded-xl" />
    </div>
  );
}

interface SummaryCardProps {
  title: string;
  value: string;
  detail: string;
  icon: ComponentType<{ className?: string }>;
}

function SummaryCard({ title, value, detail, icon: Icon }: SummaryCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        <Icon className="size-4 text-muted-foreground" aria-hidden="true" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold tracking-tight">{value}</div>
        <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

interface DailySpendChartProps {
  daily: ApiUsageDailyPoint[];
}

function DailySpendChart({ daily }: DailySpendChartProps) {
  const chart = useMemo(() => {
    const max = daily.reduce((highest, point) => Math.max(highest, point.cost), 0);
    return daily.map((point) => ({ ...point, height: max > 0 ? Math.max(4, (point.cost / max) * 100) : 4 }));
  }, [daily]);

  if (daily.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
        No daily usage in this range.
      </div>
    );
  }

  return (
    <figure className="h-64" aria-label="Daily approximate spend chart">
      <div className="flex h-52 items-end gap-1 overflow-x-auto rounded-lg border bg-muted/20 px-3 py-4">
        {chart.map((point) => (
          <Tooltip key={point.date}>
            <TooltipTrigger asChild>
              <div className="flex h-full min-w-2 flex-1 items-end justify-center" style={{ flexBasis: daily.length > 90 ? "0.75rem" : undefined }}>
                <div
                  className="w-full min-w-2 rounded-t bg-primary/80 transition-colors hover:bg-primary"
                  style={{ height: `${point.height}%` }}
                  aria-label={`${point.date}: ${formatCurrency(point.cost)}`}
                />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <div className="text-sm font-medium">{point.date}</div>
              <div className="text-xs text-muted-foreground">{formatCurrency(point.cost)}</div>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
      <div className="mt-2 flex justify-between text-xs text-muted-foreground">
        <span>{daily[0]?.date}</span>
        {daily.length === 1 ? <span>One day</span> : <span>{daily[daily.length - 1]?.date}</span>}
      </div>
    </figure>
  );
}

interface UsageFetchState {
  usage: ApiUsageResponse | null;
  status: "loading" | "ready" | "error";
  error: string | null;
}

type UsageFetchAction =
  | { type: "loading" }
  | { type: "loaded"; usage: ApiUsageResponse }
  | { type: "failed"; error: string };

function usageFetchReducer(state: UsageFetchState, action: UsageFetchAction): UsageFetchState {
  switch (action.type) {
    case "loading":
      return { ...state, status: "loading", error: null };
    case "loaded":
      return { usage: action.usage, status: "ready", error: null };
    case "failed":
      return { ...state, status: "error", error: action.error };
  }
}

function UsageTable({ rows }: { rows: ApiUsageRow[] }) {
  if (rows.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
        No usage rows match the selected filters.
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead scope="col">Day</TableHead>
          <TableHead scope="col">Project</TableHead>
          <TableHead scope="col">Root project</TableHead>
          <TableHead scope="col">Source</TableHead>
          <TableHead scope="col" className="text-right">Spend</TableHead>
          <TableHead scope="col" className="text-right">Tokens</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={`${row.date}-${row.project}-${row.rootProject}-${row.workspacePath}-${row.source}`}>
            <TableCell>{row.date}</TableCell>
            <TableCell><TruncatedName value={row.project} path={row.workspacePath} /></TableCell>
            <TableCell><TruncatedName value={row.rootProject} path={row.rootWorkspacePath} /></TableCell>
            <TableCell><TruncatedName value={row.source} className="max-w-[12rem]" /></TableCell>
            <TableCell className="text-right font-medium">
              <div>{formatCurrency(row.cost)}</div>
              <div className="text-xs text-muted-foreground">
                {row.apiEquivalentCostAvailable ? `API ${formatCurrency(row.apiEquivalentCost)}` : "API estimate unavailable"}
              </div>
            </TableCell>
            <TableCell className="text-right">
              <div>{compactTokens(row.tokens)}</div>
              <div className="text-xs text-muted-foreground">
                in {formatNumber(row.tokens.input)} · out {formatNumber(row.tokens.output)}
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function Usage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [{ usage, status, error }, dispatchUsage] = useReducer(usageFetchReducer, { usage: null, status: "loading", error: null });

  const filters = useMemo(() => ({
    from: searchParams.get("from") || defaultFromDate(),
    to: searchParams.get("to") || todayDate(),
    project: searchParams.get("project") || "",
    rootProject: searchParams.get("rootProject") || "",
  }), [searchParams]);

  const loadUsage = useCallback((refresh = false) => {
    dispatchUsage({ type: "loading" });
    const request = refresh ? api.usage.refresh(filters) : api.usage.get(filters);
    void request
      .then((loadedUsage) => dispatchUsage({ type: "loaded", usage: loadedUsage }))
      .catch((err: unknown) => {
        dispatchUsage({ type: "failed", error: err instanceof Error ? err.message : "Failed to load usage" });
      });
  }, [filters]);

  useEffect(() => {
    loadUsage();
  }, [loadUsage]);

  const updateFilter = useCallback((name: keyof typeof filters, value: string) => {
    const next = new URLSearchParams(searchParams);
    if (value) {
      next.set(name, value);
    } else {
      next.delete(name);
    }
    setSearchParams(next);
  }, [searchParams, setSearchParams]);

  const resetFilters = useCallback(() => {
    setSearchParams(new URLSearchParams());
  }, [setSearchParams]);

  const totals = usage?.totals;
  const totalTokens = totals ? tokenTotal(totals.tokens) : 0;
  const loading = status === "loading";

  return (
    <section className="mx-auto flex w-full max-w-7xl flex-col gap-5 pb-8" aria-labelledby="usage-title">
      <header className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 id="usage-title" className="text-3xl font-semibold tracking-tight">Global usage (Beta)</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            Track approximate inference spend and token activity across all SGAI workspaces.
          </p>
        </div>
        <Button variant="outline" onClick={() => loadUsage(true)} disabled={loading}>
          <RefreshCw className={cn("mr-2 size-4", loading && "animate-spin")} />
          Refresh
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[repeat(4,minmax(0,1fr))_auto] md:items-end">
          <label htmlFor="usage-filter-from" className="space-y-1 text-sm font-medium">
            <span>From</span>
            <Input id="usage-filter-from" type="date" value={filters.from} onChange={(event) => updateFilter("from", event.target.value)} />
          </label>
          <label htmlFor="usage-filter-to" className="space-y-1 text-sm font-medium">
            <span>To</span>
            <Input id="usage-filter-to" type="date" value={filters.to} onChange={(event) => updateFilter("to", event.target.value)} />
          </label>
          <label className="space-y-1 text-sm font-medium">
            <span>Project</span>
            <Select value={filters.project} onChange={(event) => updateFilter("project", event.target.value)}>
              <SelectOption value="">All projects</SelectOption>
              {(usage?.filters.projects || []).map((project) => <SelectOption key={project} value={project}>{project}</SelectOption>)}
            </Select>
          </label>
          <label className="space-y-1 text-sm font-medium">
            <span>Root project</span>
            <Select value={filters.rootProject} onChange={(event) => updateFilter("rootProject", event.target.value)}>
              <SelectOption value="">All roots</SelectOption>
              {(usage?.filters.rootProjects || []).map((root) => <SelectOption key={root} value={root}>{root}</SelectOption>)}
            </Select>
          </label>
          <Button variant="secondary" onClick={resetFilters}>Reset</Button>
        </CardContent>
      </Card>

      {error ? (
        <Alert className="border-destructive/50 text-destructive">
          <AlertCircle className="size-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {loading && !usage ? <UsageSkeleton /> : null}

      {usage ? (
        <>
          {usage.warning ? (
            <Alert>
              <AlertCircle className="size-4" />
              <AlertDescription>{usage.warning}</AlertDescription>
            </Alert>
          ) : null}

          <div className="grid gap-3 md:grid-cols-3">
            <SummaryCard title="Approximate spend" value={formatCurrency(totals?.cost || 0)} detail={`Metered report ${formatCurrency(totals?.meteredReportedCost || 0)}`} icon={Coins} />
            <SummaryCard title="API-equivalent spend" value={totals?.apiEquivalentCostAvailable ? formatCurrency(totals.apiEquivalentCost) : "Unavailable"} detail="Based on model pricing metadata when available" icon={Sigma} />
            <SummaryCard title="Total tokens" value={formatNumber(totalTokens)} detail={`Input ${formatNumber(totals?.tokens.input || 0)} · output ${formatNumber(totals?.tokens.output || 0)}`} icon={Database} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Daily spend</CardTitle>
            </CardHeader>
            <CardContent>
              <DailySpendChart daily={usage.daily} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Grouped usage rows</CardTitle>
            </CardHeader>
            <CardContent>
              <UsageTable rows={usage.rows} />
            </CardContent>
          </Card>
        </>
      ) : null}
    </section>
  );
}
