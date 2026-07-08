import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Cog, Loader2, Play, RotateCcw, Square, ScrollText } from "lucide-react";
import { api, type ServiceView, type JobResponse } from "~/lib/api";
import { useJobStream, useEventSource } from "~/lib/sse";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";

function LogsDialog({
  service,
  open,
  onOpenChange,
}: {
  service: string | null;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const [lines, setLines] = useState<string[]>([]);
  useEventSource(service ? `/api/logs/${encodeURIComponent(service)}?tail=80&follow=1` : null, (line) => {
    setLines((prev) => {
      const next = [...prev, line];
      return next.length > 500 ? next.slice(next.length - 500) : next;
    });
  });
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>logs · {service}</DialogTitle>
          <DialogDescription>Stream en vivo de docker logs</DialogDescription>
        </DialogHeader>
        <pre className="max-h-[60vh] overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs whitespace-pre-wrap">
          {lines.join("\n") || "Esperando salida…"}
        </pre>
      </DialogContent>
    </Dialog>
  );
}

export default function ServicesRoute() {
  const qc = useQueryClient();
  const { data, isLoading, error } = useQuery<ServiceView[]>({
    queryKey: ["services"],
    queryFn: () => api<ServiceView[]>("/api/services"),
    refetchInterval: 5_000,
  });
  const [logsSvc, setLogsSvc] = useState<string | null>(null);
  const [errMsg, setErrMsg] = useState<string | null>(null);

  const ctrlMut = useMutation({
    mutationFn: ({ name, verb }: { name: string; verb: "up" | "stop" | "restart" }) =>
      api<JobResponse>(`/api/services/${encodeURIComponent(name)}/${verb}`, { method: "POST" }),
    onSuccess: (res, vars) => {
      toast.success(`${vars.verb} ${vars.name}`, { description: `job ${res.job_id}` });
      qc.invalidateQueries({ queryKey: ["services"] });
      qc.invalidateQueries({ queryKey: ["status"] });
    },
    onError: (err: unknown, vars) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error(`${vars.verb} ${vars.name} falló`, { description: msg });
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Servicios docker</h1>
          <p className="text-sm text-muted-foreground">
            Arranca, para o reinicia cada servicio del compose.
          </p>
        </div>
        <Badge variant="outline">
          <Cog className="mr-1 h-3 w-3" /> {data?.length ?? 0} servicios
        </Badge>
      </div>
      {errMsg && (
        <Card className="border-destructive/40">
          <CardContent className="py-3 text-sm text-destructive">{errMsg}</CardContent>
        </Card>
      )}
      {error && (
        <Card>
          <CardContent className="py-3 text-sm text-destructive">
            {(error as Error).message}
          </CardContent>
        </Card>
      )}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Servicios definidos</CardTitle>
          <CardDescription>
            {isLoading ? "Cargando…" : "Acciones inmediatas vía docker compose."}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-2">
          {data?.map((svc) => (
            <motion.div
              layout
              key={svc.name}
              className="flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2"
            >
              <div className="flex items-center gap-3">
                <span className="font-mono text-sm">{svc.name}</span>
                <Badge variant={svc.state === "running" ? "success" : "secondary"}>
                  {svc.state}
                </Badge>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={ctrlMut.isPending}
                  onClick={() => ctrlMut.mutate({ name: svc.name, verb: "up" })}
                >
                  <Play className="h-3 w-3" /> up
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={ctrlMut.isPending}
                  onClick={() => ctrlMut.mutate({ name: svc.name, verb: "stop" })}
                >
                  <Square className="h-3 w-3" /> stop
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={ctrlMut.isPending}
                  onClick={() => ctrlMut.mutate({ name: svc.name, verb: "restart" })}
                >
                  {ctrlMut.isPending && ctrlMut.variables?.name === svc.name ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <RotateCcw className="h-3 w-3" />
                  )}
                  restart
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setLogsSvc(svc.name)}>
                  <ScrollText className="h-3 w-3" /> logs
                </Button>
              </div>
            </motion.div>
          ))}
          {data?.length === 0 && (
            <p className="text-sm text-muted-foreground">Sin servicios en el compose.</p>
          )}
        </CardContent>
      </Card>
      {logsSvc && (
        <LogsDialog
          service={logsSvc}
          open
          onOpenChange={(next) => {
            if (!next) setLogsSvc(null);
          }}
        />
      )}
    </div>
  );
}