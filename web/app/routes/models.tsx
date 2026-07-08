import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Download, Loader2, Trash2, Database } from "lucide-react";
import { api, type ModelView, type JobResponse } from "~/lib/api";
import { useJobStream } from "~/lib/sse";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { Input } from "~/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";

function PullDialog({
  tag,
  jobId,
  open,
  onOpenChange,
}: {
  tag: string | null;
  jobId: string | null;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const job = useJobStream(jobId);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>pull {tag ?? ""}</DialogTitle>
          <DialogDescription>job {jobId ?? "—"}</DialogDescription>
        </DialogHeader>
        <div className="max-h-80 overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs">
          {job.lines.length === 0 && <span className="text-muted-foreground">Iniciando…</span>}
          {job.lines.map((l, i) => (
            <div
              key={i}
              className={l.stream === "stderr" ? "text-amber-300" : "text-foreground/90"}
            >
              {l.text}
            </div>
          ))}
          {job.done && (
            <div className={job.ok ? "mt-2 text-emerald-300" : "mt-2 text-red-300"}>
              {job.ok ? "✓ listo" : `✗ ${job.error ?? "falló"}`}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default function ModelsRoute() {
  const qc = useQueryClient();
  const { data, isLoading, error } = useQuery<ModelView[]>({
    queryKey: ["models"],
    queryFn: () => api<ModelView[]>("/api/models"),
    refetchInterval: 5_000,
  });
  const [tag, setTag] = useState("");
  const [activeJob, setActiveJob] = useState<{ tag: string; id: string } | null>(null);
  const [errMsg, setErrMsg] = useState<string | null>(null);

  const pullMut = useMutation({
    mutationFn: (t: string) =>
      api<JobResponse>("/api/models/pull", { method: "POST", body: { tag: t } }),
    onSuccess: (res, t) => {
      setActiveJob({ tag: t, id: res.job_id });
      toast.success(`pull ${t} encolado`, { description: `job ${res.job_id}` });
      qc.invalidateQueries({ queryKey: ["models"] });
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error("pull falló", { description: msg });
    },
  });

  const rmMut = useMutation({
    mutationFn: (t: string) =>
      api<JobResponse>("/api/models/rm", { method: "POST", body: { tag: t } }),
    onSuccess: (res, t) => {
      toast.success(`rm ${t} encolado`, { description: `job ${res.job_id}` });
      qc.invalidateQueries({ queryKey: ["models"] });
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error("rm falló", { description: msg });
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Modelos Ollama</h1>
          <p className="text-sm text-muted-foreground">
            Descarga o elimina modelos del contenedor mcp-tools-ollama.
          </p>
        </div>
        <form
          className="flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            const t = tag.trim();
            if (!t) return;
            pullMut.mutate(t);
            setTag("");
          }}
        >
          <Input
            value={tag}
            onChange={(e) => setTag(e.target.value)}
            placeholder="qwen2.5:7b"
            className="w-44"
          />
          <Button type="submit" disabled={pullMut.isPending || tag.trim() === ""}>
            {pullMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
            Pull
          </Button>
        </form>
      </div>
      {errMsg && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{errMsg}</AlertDescription>
        </Alert>
      )}
      {error && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>
            {(error as Error).message ?? "no se pudo cargar /api/models"}
          </AlertDescription>
        </Alert>
      )}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Database className="h-4 w-4" />
            Instalados
          </CardTitle>
          <CardDescription>
            {isLoading
              ? "Cargando…"
              : `${data?.length ?? 0} modelo${data?.length === 1 ? "" : "s"}`}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-2">
          {data?.length ? (
            data.map((m) => (
              <motion.div
                layout
                key={m.tag}
                className="flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2"
              >
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm">{m.tag}</span>
                  <Badge variant="outline">{m.size}</Badge>
                  <span className="text-xs text-muted-foreground">{m.modified}</span>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={rmMut.isPending}
                  onClick={() => rmMut.mutate(m.tag)}
                >
                  {rmMut.isPending && rmMut.variables === m.tag ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <Trash2 className="h-3 w-3" />
                  )}
                  rm
                </Button>
              </motion.div>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">Sin modelos instalados.</p>
          )}
        </CardContent>
      </Card>
      {activeJob && (
        <PullDialog
          tag={activeJob.tag}
          jobId={activeJob.id}
          open
          onOpenChange={(next) => {
            if (!next) setActiveJob(null);
          }}
        />
      )}
    </div>
  );
}