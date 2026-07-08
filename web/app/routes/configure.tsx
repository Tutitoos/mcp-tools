import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Layers, Loader2, Save } from "lucide-react";
import { api, type ToolView, type JobResponse } from "~/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Switch } from "~/components/ui/switch";
import { Badge } from "~/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";

export default function ConfigureRoute() {
  const qc = useQueryClient();
  const { data: tools, isLoading } = useQuery<ToolView[]>({
    queryKey: ["tools"],
    queryFn: () => api<ToolView[]>("/api/tools"),
  });
  const [selected, setSelected] = useState<Set<string> | null>(null);
  const [error, setError] = useState<string | null>(null);

  // initialise selection from server data
  const chosen = useMemo(() => {
    if (selected) return selected;
    if (!tools) return new Set<string>();
    return new Set(tools.filter((t) => t.selected).map((t) => t.key));
  }, [tools, selected]);

  function toggle(key: string) {
    setSelected(new Set(chosen));
    const next = new Set(chosen);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setSelected(next);
  }

  const mutate = useMutation({
    mutationFn: () =>
      api<JobResponse>("/api/configure", {
        method: "POST",
        body: { selected: Array.from(chosen) },
      }),
    onSuccess: (res) => {
      toast.success("Configuración encolada", {
        description: `job ${res.job_id}`,
      });
      qc.invalidateQueries({ queryKey: ["tools"] });
      qc.invalidateQueries({ queryKey: ["status"] });
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
      toast.error("No se pudo aplicar la selección", { description: msg });
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Configurar selección</h1>
          <p className="text-sm text-muted-foreground">
            Marca o desmarca los componentes. El orquestador respeta las dependencias declaradas.
          </p>
        </div>
        <Button
          disabled={mutate.isPending || isLoading || !tools}
          onClick={() => mutate.mutate()}
        >
          {mutate.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          Aplicar cambios
        </Button>
      </div>
      {error && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Layers className="h-4 w-4" />
            Selección actual
          </CardTitle>
          <CardDescription>
            {chosen.size} tool{chosen.size === 1 ? "" : "s"} marcados
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-2">
          {isLoading || !tools ? (
            <div className="text-sm text-muted-foreground">Cargando…</div>
          ) : (
            tools.map((t) => (
              <motion.div
                key={t.key}
                layout
                className="flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2"
              >
                <div className="flex items-center gap-3">
                  <Switch checked={chosen.has(t.key)} onCheckedChange={() => toggle(t.key)} />
                  <div>
                    <div className="flex items-center gap-2 text-sm">
                      <span className="font-mono">{t.key}</span>
                      <Badge variant="outline">{t.deploy}</Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">{t.summary}</p>
                  </div>
                </div>
                {(t.deps?.length ?? 0) > 0 && (
                  <span className="text-xs text-muted-foreground">
                    deps: {t.deps.join(", ")}
                  </span>
                )}
              </motion.div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}