import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion, AnimatePresence } from "motion/react";
import { toast } from "sonner";
import {
  AlertCircle,
  Check,
  Loader2,
  RefreshCcw,
  Trash2,
  Download,
  ScrollText,
} from "lucide-react";
import { api, type ToolView, type JobResponse } from "~/lib/api";
import { Link } from "react-router";
import { useJobStream } from "~/lib/sse";
import { Card, CardContent, CardHeader, CardTitle } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { SkeletonRow } from "~/components/ui/skeleton";
import { Modal, JobLogPane } from "~/components/modal";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";

type Action = "install" | "upgrade" | "uninstall";

function statusVariant(
  view: ToolView,
): "success" | "warning" | "destructive" | "secondary" {
  if (view.installed) return "success";
  if (view.deploy === "Sudo") return "warning";
  return "secondary";
}

function statusLabel(view: ToolView): string {
  if (view.installed && view.version) return `v${view.version}`;
  if (view.installed) return "instalado";
  if (view.deploy === "Sudo") return "requiere sudo";
  return "no instalado";
}

function runAction(
  action: Action,
  key: string,
  body?: Record<string, unknown>,
) {
  const path = `/api/tools/${encodeURIComponent(key)}/${action}`;
  return api<JobResponse>(path, { method: "POST", body });
}

function RunDialog({
  toolKey,
  toolLabel,
  action,
  jobId,
  open,
  onOpenChange,
}: {
  toolKey: string;
  toolLabel: string;
  action: Action;
  jobId: string | null;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const job = useJobStream(jobId);
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      title={`${action} · ${toolKey}`}
      description={`${toolLabel} · job ${jobId ?? "—"}`}
    >
      <JobLogPane
        lines={job.lines}
        done={job.done}
        ok={job.ok}
        error={job.error}
      />
    </Modal>
  );
}

function ToolRow({ view }: { view: ToolView }) {
  const qc = useQueryClient();
  const [pending, setPending] = useState<Action | null>(null);
  const [jobId, setJobId] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const mutate = useMutation({
    mutationFn: (vars: { action: Action; body?: Record<string, unknown> }) =>
      runAction(vars.action, view.key, vars.body),
    onSuccess: (res, vars) => {
      setPending(null);
      setJobId(res.job_id);
      setDialogOpen(true);
      toast.success(`${vars.action} ${view.key} encolado`, {
        description: `job ${res.job_id}`,
      });
    },
    onError: (err: unknown, vars) => {
      setPending(null);
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
      toast.error(`No se pudo ${vars.action} ${view.key}`, {
        description: msg,
      });
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["tools"] });
      qc.invalidateQueries({ queryKey: ["status"] });
    },
  });

  function start(action: Action, body?: Record<string, unknown>) {
    setError(null);
    setPending(action);
    mutate.mutate({ action, body });
  }

  return (
    <motion.div layout>
      <div className="row-vc grid gap-4 px-4 py-4 md:grid-cols-[1fr_auto] md:items-center">
          <div className="min-w-0 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-sm font-semibold">
                {view.key}
              </span>
              <Badge variant={statusVariant(view)}>{statusLabel(view)}</Badge>
              <Badge variant="outline">{view.deploy}</Badge>
              <Badge variant={view.selected ? "default" : "outline"}>
                {view.selected ? "selected" : "unselected"}
              </Badge>
            </div>
            <p className="text-sm font-medium">{view.label}</p>
            <p className="text-xs text-muted-foreground">{view.summary}</p>
            {error && (
              <Alert variant="destructive" className="py-2">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle className="text-xs">Error</AlertTitle>
                <AlertDescription className="text-xs">{error}</AlertDescription>
              </Alert>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2 md:justify-end">
            <Button
              size="sm"
              variant="outline"
              disabled={pending !== null || view.installed}
              onClick={() => start("install")}
            >
              {pending === "install" ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : view.installed ? (
                <Check className="h-3 w-3" />
              ) : (
                <Download className="h-3 w-3" />
              )}
              {view.installed ? "instalado" : "install"}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={pending !== null || !view.installed}
              onClick={() => start("upgrade")}
            >
              {pending === "upgrade" ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <RefreshCcw className="h-3 w-3" />
              )}
              upgrade
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={pending !== null || !view.installed}
              onClick={() => start("uninstall", { force: false })}
            >
              {pending === "uninstall" ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Trash2 className="h-3 w-3" />
              )}
              uninstall
            </Button>
            <Button size="sm" variant="ghost" asChild>
              <Link to={`/jobs?q=${encodeURIComponent(view.key)}`}>
                <ScrollText className="h-3 w-3" />
                logs
              </Link>
            </Button>
          </div>
      </div>
      <AnimatePresence>
        {dialogOpen && (
          <RunDialog
            toolKey={view.key}
            toolLabel={view.label}
            action={pending ?? "install"}
            jobId={jobId}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
          />
        )}
      </AnimatePresence>
    </motion.div>
  );
}

export default function ToolsRoute() {
  const { data, isLoading, error } = useQuery<ToolView[]>({
    queryKey: ["tools"],
    queryFn: () => api<ToolView[]>("/api/tools"),
    refetchInterval: 5_000,
  });
  return (
    <div className="space-y-6">
      {isLoading && (
        <div className="grid gap-2">
          {[0, 1, 2, 3].map((i) => (
            <SkeletonRow key={i} />
          ))}
        </div>
      )}
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>
            {(error as Error).message ?? "no se pudo cargar /api/tools"}
          </AlertDescription>
        </Alert>
      )}
      {data && (
        <div className="grid gap-3">
          {data.length === 0 ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">
                  Sin tools en el registro
                </CardTitle>
              </CardHeader>
            </Card>
          ) : (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0">
                <CardTitle className="text-base">Registro de tools</CardTitle>
                <Badge variant="outline">{data.length} totales</Badge>
              </CardHeader>
              <CardContent className="grid gap-3">
                {data.map((v) => (
                  <ToolRow key={v.key} view={v} />
                ))}
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}
