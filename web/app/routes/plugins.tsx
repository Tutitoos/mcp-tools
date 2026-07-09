import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion, AnimatePresence } from "motion/react";
import { toast } from "sonner";
import {
  AlertCircle,
  Link,
  Loader2,
  Pause,
  Play,
  Unlink,
  ScrollText,
} from "lucide-react";
import { api, type PluginView, type JobResponse } from "~/lib/api";
import { Link as RouterLink } from "react-router";
import { useJobStream } from "~/lib/sse";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { SkeletonRow } from "~/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";

type Action = "link" | "unlink" | "enable" | "disable";

const VERB_LABEL: Record<Action, string> = {
  link: "Link",
  unlink: "Unlink",
  enable: "Enable",
  disable: "Disable",
};

function runAction(action: Action, name: string) {
  const path = `/api/plugins/${encodeURIComponent(name)}/${action}`;
  return api<JobResponse>(path, { method: "POST" });
}

function RunDialog({
  pluginName,
  action,
  jobId,
  open,
  onOpenChange,
}: {
  pluginName: string;
  action: Action;
  jobId: string | null;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const qc = useQueryClient();
  const job = useJobStream(jobId);

  // On job completion: refetch the plugins list (only when the action
  // actually succeeded — a failed job leaves lockfile state unchanged)
  // and surface a toast either way.
  useEffect(() => {
    if (!job.done) return;
    if (job.ok) {
      qc.invalidateQueries({ queryKey: ["plugins"] });
      toast.success(`${VERB_LABEL[action]} ${pluginName} completado`);
    } else {
      toast.error(`${VERB_LABEL[action]} ${pluginName} falló`, {
        description: job.error ?? undefined,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [job.done]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            Ejecutando <code>{VERB_LABEL[action]}</code> en{" "}
            <code>{pluginName}</code>
          </DialogTitle>
          <DialogDescription>job {jobId ?? "—"}</DialogDescription>
        </DialogHeader>
        <div className="max-h-80 overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs">
          {job.lines.length === 0 && job.open && (
            <p className="text-muted-foreground">Iniciando…</p>
          )}
          {job.lines.map((l, i) => (
            <div
              key={i}
              className={
                l.stream === "stderr" ? "text-warning" : "text-foreground/90"
              }
            >
              {l.text}
            </div>
          ))}
          {job.done && (
            <div
              className={job.ok ? "mt-2 text-success" : "mt-2 text-destructive"}
            >
              {job.ok ? "✓ completado" : `✗ ${job.error ?? "falló"}`}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cerrar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PluginRow({ view }: { view: PluginView }) {
  const [jobId, setJobId] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const mutate = useMutation({
    mutationFn: (action: Action) => runAction(action, view.name),
    onSuccess: (res, action) => {
      setJobId(res.job_id);
      setDialogOpen(true);
      toast.success(`${VERB_LABEL[action]} ${view.name} encolado`, {
        description: `job ${res.job_id}`,
      });
    },
    onError: (err: unknown, action) => {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
      toast.error(`No se pudo ${VERB_LABEL[action]} ${view.name}`, {
        description: msg,
      });
    },
  });

  function start(action: Action) {
    setError(null);
    mutate.mutate(action);
  }

  const busy = mutate.isPending;
  const dialogAction = mutate.variables ?? "link";

  function actionButton(action: Action, icon: React.ReactNode) {
    return (
      <Button
        size="sm"
        variant="outline"
        disabled={busy}
        onClick={() => start(action)}
      >
        {busy && mutate.variables === action ? (
          <Loader2 className="h-3 w-3 animate-spin" />
        ) : (
          icon
        )}
        {VERB_LABEL[action]}
      </Button>
    );
  }

  return (
    <motion.div layout>
      <Card className="border-border/60">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="font-mono text-sm">{view.name}</CardTitle>
          <Badge variant="outline">v{view.version || "0.0.0"}</Badge>
        </CardHeader>
        <CardContent className="space-y-2 pt-0">
          <div className="flex flex-wrap items-center gap-2">
            {view.linked ? (
              <Badge variant="success">Linked</Badge>
            ) : (
              <Badge variant="secondary">Available</Badge>
            )}
            {view.linked &&
              (view.enabled ? (
                <Badge variant="success">Enabled</Badge>
              ) : (
                <Badge variant="warning">Disabled</Badge>
              ))}
          </div>
          <CardDescription>{view.description}</CardDescription>
          {view.extensions.length === 0 ? (
            <p className="text-xs text-muted-foreground">
              sin extensiones declaradas
            </p>
          ) : (
            <ul className="space-y-0.5">
              {view.extensions.map((ext) => (
                <li
                  key={ext}
                  className="font-mono text-xs text-muted-foreground"
                >
                  {ext}
                </li>
              ))}
            </ul>
          )}
          <div className="font-mono text-xs text-muted-foreground">
            {view.path}
          </div>
          {error && (
            <Alert variant="destructive" className="py-2">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle className="text-xs">Error</AlertTitle>
              <AlertDescription className="text-xs">{error}</AlertDescription>
            </Alert>
          )}
        </CardContent>
        <CardFooter className="justify-end gap-2">
          {!view.linked && actionButton("link", <Link className="h-3 w-3" />)}
          {view.linked &&
            view.enabled &&
            actionButton("disable", <Pause className="h-3 w-3" />)}
          {view.linked &&
            !view.enabled &&
            actionButton("enable", <Play className="h-3 w-3" />)}
          {view.linked &&
            actionButton("unlink", <Unlink className="h-3 w-3" />)}
          <Button size="sm" variant="ghost" asChild>
            <RouterLink to={`/jobs?q=${encodeURIComponent(view.name)}`}>
              <ScrollText className="h-3 w-3" />
              logs
            </RouterLink>
          </Button>
        </CardFooter>
      </Card>
      <AnimatePresence>
        {dialogOpen && (
          <RunDialog
            pluginName={view.name}
            action={dialogAction}
            jobId={jobId}
            open={dialogOpen}
            onOpenChange={setDialogOpen}
          />
        )}
      </AnimatePresence>
    </motion.div>
  );
}

export default function PluginsRoute() {
  const { data, isLoading, error } = useQuery<PluginView[]>({
    queryKey: ["plugins"],
    queryFn: () => api<PluginView[]>("/api/plugins"),
    refetchInterval: 10_000,
  });
  return (
    <div className="space-y-6">
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
            {(error as Error).message ?? "no se pudo cargar /api/plugins"}
          </AlertDescription>
        </Alert>
      )}
      {data &&
        (data.length === 0 ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Sin plugins en el workspace
              </CardTitle>
              <CardDescription>
                Crea un subdirectorio en <code>~/mcp-tools/plugins/</code> con
                su propio <code>package.json</code>.
              </CardDescription>
            </CardHeader>
          </Card>
        ) : (
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0">
              <CardTitle className="text-base">
                Plugins del workspace
              </CardTitle>
              <Badge variant="outline">{data.length} totales</Badge>
            </CardHeader>
            <CardContent className="grid gap-3">
              {data.map((v) => (
                <PluginRow key={v.name} view={v} />
              ))}
            </CardContent>
          </Card>
        ))}
    </div>
  );
}
