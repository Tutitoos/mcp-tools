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
  CardHeader,
  CardTitle,
} from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { SkeletonRow } from "~/components/ui/skeleton";
import { Modal, JobLogPane } from "~/components/modal";
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
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      title={
        <>
          Ejecutando <code>{VERB_LABEL[action]}</code> en{" "}
          <code>{pluginName}</code>
        </>
      }
      description={`job ${jobId ?? "—"}`}
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
      <div className="row-vc space-y-2 px-4 py-4">
        <div className="flex flex-row items-center justify-between">
          <span className="font-mono text-sm font-semibold">{view.name}</span>
          <Badge variant="outline">v{view.version || "0.0.0"}</Badge>
        </div>
        <div className="space-y-2">
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
          <p className="text-sm text-muted-foreground">{view.description}</p>
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
        </div>
        <div className="flex justify-end gap-2 pt-1">
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
        </div>
      </div>
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
