import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { Activity, AlertCircle, Ban, Copy, Loader2 } from "lucide-react";
import { api, type JobSummary } from "~/lib/api";
import { useJobStream } from "~/lib/sse";
import { Input } from "~/components/ui/input";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { SkeletonRow } from "~/components/ui/skeleton";

const RELTIME = new Intl.RelativeTimeFormat("es", { numeric: "auto" });
function relTime(iso: string): string {
  const then = new Date(iso).getTime();
  const now = Date.now();
  const s = Math.round((then - now) / 1000);
  const abs = Math.abs(s);
  if (abs < 60) return RELTIME.format(s, "second");
  if (abs < 3600) return RELTIME.format(Math.round(s / 60), "minute");
  return RELTIME.format(Math.round(s / 3600), "hour");
}

function statusBadge(status: JobSummary["status"]) {
  switch (status) {
    case "running":
      return <Badge variant="secondary">running</Badge>;
    case "ok":
      return <Badge variant="success">ok</Badge>;
    case "error":
      return <Badge variant="destructive">error</Badge>;
  }
}

function JobRow({
  job,
  selected,
  onSelect,
}: {
  job: JobSummary;
  selected: boolean;
  onSelect: () => void;
}) {
  const qc = useQueryClient();
  const cancel = useMutation({
    mutationFn: () => api(`/api/jobs/${job.id}/cancel`, { method: "POST" }),
    onSettled: () => qc.invalidateQueries({ queryKey: ["jobs"] }),
  });
  const duration =
    job.finished_at != null
      ? `${(
          (new Date(job.finished_at).getTime() -
            new Date(job.started_at).getTime()) /
          1000
        ).toFixed(1)}s`
      : null;
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`w-full rounded-md border px-3 py-2 text-left transition-colors ${
        selected
          ? "border-primary/60 bg-primary/5"
          : "border-border/60 hover:bg-accent/40"
      }`}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="font-mono text-xs text-muted-foreground">
          {job.id.slice(0, 8)}
        </span>
        {statusBadge(job.status)}
      </div>
      <div className="mt-1 truncate text-sm">{job.label}</div>
      <div className="mt-1 flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>
          {relTime(job.started_at)}
          {duration ? ` · ${duration}` : ""}
        </span>
        {job.status === "running" && (
          <Button
            size="icon"
            variant="ghost"
            className="h-6 w-6"
            disabled={cancel.isPending}
            onClick={(e) => {
              e.stopPropagation();
              cancel.mutate();
            }}
          >
            {cancel.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <Ban className="h-3 w-3" />
            )}
          </Button>
        )}
      </div>
    </button>
  );
}

export default function JobsRoute() {
  const { data, isLoading, error } = useQuery<JobSummary[]>({
    queryKey: ["jobs"],
    queryFn: () => api<JobSummary[]>("/api/jobs"),
    refetchInterval: 3_000,
  });
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const filtered = useMemo(() => {
    if (!q) return data ?? [];
    const needle = q.toLowerCase();
    return (data ?? []).filter((j) => j.label.toLowerCase().includes(needle));
  }, [data, q]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selectedJob = data?.find((j) => j.id === selectedId) ?? null;
  const job = useJobStream(selectedId);

  useEffect(() => {
    if (q && !selectedId && filtered.length > 0) {
      setSelectedId(filtered[0].id);
    }
  }, [q, filtered, selectedId]);

  function copyLog() {
    navigator.clipboard
      .writeText(job.lines.map((l) => l.text).join("\n"))
      .then(() => toast.success("log copiado"));
  }

  function setFilter(next: string) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next) p.set("q", next);
        else p.delete("q");
        return p;
      },
      { replace: true },
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-[minmax(280px,340px)_1fr]">
      <Card className="flex flex-col">
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="flex items-center gap-2 text-base">
            <Activity className="h-4 w-4" />
            Jobs
          </CardTitle>
          <Badge variant="outline">{filtered.length}</Badge>
        </CardHeader>
        <CardContent className="space-y-2">
          <Input
            placeholder="Filtrar por label (tool/plugin)…"
            value={q}
            onChange={(e) => setFilter(e.target.value)}
            className="mb-1"
          />
          {isLoading && (
            <div className="grid gap-2">
              {[0, 1, 2].map((i) => (
                <SkeletonRow key={i} />
              ))}
            </div>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                {(error as Error).message ?? "no se pudo cargar /api/jobs"}
              </AlertDescription>
            </Alert>
          )}
          {data && data.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sin jobs. Ejecuta cualquier acción (link/install/restart) para
              ver aquí su progreso.
            </p>
          )}
          {data && data.length > 0 && filtered.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sin jobs que coincidan con "{q}".
            </p>
          )}
          {filtered.map((j) => (
            <JobRow
              key={j.id}
              job={j}
              selected={j.id === selectedId}
              onSelect={() => setSelectedId(j.id)}
            />
          ))}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Log · {selectedJob?.label ?? "—"}
          </CardTitle>
          {selectedId && (
            <CardDescription className="font-mono text-xs">
              job {selectedId}
            </CardDescription>
          )}
        </CardHeader>
        <CardContent>
          {!selectedId ? (
            <p className="text-sm text-muted-foreground">
              Selecciona un job para ver su log.
            </p>
          ) : (
            <div className="space-y-3">
              <div className="max-h-[70vh] overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs">
                {job.lines.length === 0 && job.open && (
                  <p className="text-muted-foreground">Esperando salida…</p>
                )}
                {job.lines.map((l, i) => (
                  <div
                    key={i}
                    className={
                      l.stream === "stderr"
                        ? "text-warning"
                        : "text-foreground/90"
                    }
                  >
                    {l.text}
                  </div>
                ))}
                {job.done && (
                  <div
                    className={
                      job.ok ? "mt-2 text-success" : "mt-2 text-destructive"
                    }
                  >
                    {job.ok ? "✓ completado" : `✗ ${job.error ?? "falló"}`}
                  </div>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={copyLog}
                disabled={job.lines.length === 0}
              >
                <Copy className="h-3 w-3" /> Copiar log
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
