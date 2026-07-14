import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { toast } from "sonner";
import {
  Cog,
  Loader2,
  Play,
  RotateCcw,
  Square,
  ScrollText,
} from "lucide-react";
import { api, type ServiceView, type JobResponse } from "~/lib/api";
import { useEventSource, type JobLine } from "~/lib/sse";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { SkeletonRow } from "~/components/ui/skeleton";
import { Modal, JobLogPane } from "~/components/modal";

function LogsDialog({
  service,
  open,
  onOpenChange,
}: {
  service: string | null;
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const [lines, setLines] = useState<JobLine[]>([]);
  useEventSource(
    service
      ? `/api/logs/${encodeURIComponent(service)}?tail=80&follow=1`
      : null,
    (evt) => {
      setLines((prev) => {
        const next = [...prev, evt];
        return next.length > 500 ? next.slice(next.length - 500) : next;
      });
    },
  );
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      size="xl"
      title={`logs · ${service}`}
      description="Stream en vivo de docker logs"
    >
      <JobLogPane
        lines={lines}
        waiting="Esperando salida…"
        className="max-h-[60vh] overflow-x-auto whitespace-pre-wrap"
      />
    </Modal>
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
    mutationFn: ({
      name,
      verb,
    }: {
      name: string;
      verb: "up" | "stop" | "restart";
    }) =>
      api<JobResponse>(`/api/services/${encodeURIComponent(name)}/${verb}`, {
        method: "POST",
      }),
    onSuccess: (res, vars) => {
      toast.success(`${vars.verb} ${vars.name}`, {
        description: `job ${res.job_id}`,
      });
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
      {errMsg && (
        <Card className="border-destructive/40">
          <CardContent className="py-3 text-sm text-destructive">
            {errMsg}
          </CardContent>
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
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle className="text-base">Servicios definidos</CardTitle>
            <CardDescription>
              Acciones inmediatas vía docker compose.
            </CardDescription>
          </div>
          <Badge variant="outline">
            <Cog className="mr-1 h-3 w-3" /> {data?.length ?? 0} servicios
          </Badge>
        </CardHeader>
        <CardContent className="grid gap-2">
          {isLoading
            ? [0, 1, 2].map((i) => <SkeletonRow key={i} />)
            : data?.map((svc) => (
                <motion.div
                  layout
                  key={svc.name}
                  className="row-vc flex flex-wrap items-center justify-between gap-2 px-3 py-2"
                >
                  <div className="flex min-w-0 flex-1 items-center gap-3">
                    <span className="truncate font-mono text-sm">
                      {svc.name}
                    </span>
                    <Badge
                      variant={
                        svc.state === "running" ? "success" : "secondary"
                      }
                    >
                      {svc.state}
                    </Badge>
                  </div>
                  <div className="flex shrink-0 flex-wrap items-center gap-1">
                    {(() => {
                      const isPending =
                        ctrlMut.isPending &&
                        ctrlMut.variables?.name === svc.name;
                      return (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isPending}
                            onClick={() =>
                              ctrlMut.mutate({ name: svc.name, verb: "up" })
                            }
                          >
                            {isPending ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <Play className="h-3 w-3" />
                            )}{" "}
                            up
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isPending}
                            onClick={() =>
                              ctrlMut.mutate({ name: svc.name, verb: "stop" })
                            }
                          >
                            {isPending ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <Square className="h-3 w-3" />
                            )}{" "}
                            stop
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isPending}
                            onClick={() =>
                              ctrlMut.mutate({
                                name: svc.name,
                                verb: "restart",
                              })
                            }
                          >
                            {isPending ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <RotateCcw className="h-3 w-3" />
                            )}{" "}
                            restart
                          </Button>
                        </>
                      );
                    })()}
                    <Button
                      size="icon"
                      variant="ghost"
                      onClick={() => setLogsSvc(svc.name)}
                      aria-label={`logs ${svc.name}`}
                    >
                      <ScrollText className="h-3 w-3" />
                    </Button>
                  </div>
                </motion.div>
              ))}
          {data?.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sin servicios en el compose.
            </p>
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
