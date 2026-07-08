import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Save, RefreshCcw, Settings as SettingsIcon } from "lucide-react";
import { api, type StatusPayload, type JobResponse } from "~/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Label } from "~/components/ui/label";
import { Separator } from "~/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "~/components/ui/tabs";

const ENV_KEYS = [
  "HOST_HOME",
  "HOST_UID",
  "HOST_GID",
  "MCP_TOOLS_ROOT",
  "MCP_TOOLS_DATA",
  "MCP_TOOLS_BIND",
  "MEM0_USER_ID",
] as const;

const MEM0_KEYS = [
  "MEM0_PROVIDER",
  "MEM0_LLM_MODEL",
  "MEM0_EMBED_PROVIDER",
  "MEM0_EMBED_MODEL",
  "MEM0_OLLAMA_URL",
  "MEM0_QDRANT_URL",
  "MEM0_COLLECTION",
  "MEM0_ENABLE_GRAPH",
  "MEM0_HISTORY_DB_PATH",
  "MEM0_OLLAMA_THINK",
] as const;

type Pair = [string, string];

function EnvTable({
  title,
  keys,
  data,
  onSubmit,
}: {
  title: string;
  keys: readonly string[];
  data: Record<string, string> | undefined;
  onSubmit: (values: Record<string, string>) => Promise<unknown>;
}) {
  const initial: Record<string, string> = {};
  for (const k of keys) initial[k] = data?.[k] ?? "";
  const [values, setValues] = useState<Record<string, string>>(initial);
  const [busy, setBusy] = useState(false);
  // re-sync when data reloads
  if (data && !busy && JSON.stringify(values) !== JSON.stringify(initial) && Object.values(values).every((v) => v === "" || data[k])) {
    // safe to leave as user-typed
  }

  async function submit() {
    setBusy(true);
    try {
      await onSubmit(values);
      toast.success(`${title} guardado`);
    } catch (err) {
      toast.error(`No se pudo guardar ${title}`, {
        description: err instanceof Error ? err.message : String(err),
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid gap-3">
      {keys.map((k) => (
        <div key={k} className="grid gap-1">
          <Label htmlFor={k}>{k}</Label>
          <Input
            id={k}
            value={values[k] ?? ""}
            onChange={(e) => setValues({ ...values, [k]: e.target.value })}
          />
        </div>
      ))}
      <Button onClick={submit} disabled={busy} className="justify-self-start">
        <Save className="h-4 w-4" /> Guardar {title}
      </Button>
    </div>
  );
}

export default function SettingsRoute() {
  const qc = useQueryClient();
  const { data } = useQuery<StatusPayload>({
    queryKey: ["status"],
    queryFn: () => api<StatusPayload>("/api/status"),
  });

  const syncMut = useMutation({
    mutationFn: (path: string) => api<JobResponse>(path, { method: "POST" }),
    onSuccess: (res, path) => {
      toast.success(`sync ${path}`, { description: `job ${res.job_id}` });
      qc.invalidateQueries({ queryKey: ["status"] });
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
          <p className="text-sm text-muted-foreground">
            Edita los archivos .env y lanza los sincronizadores.
          </p>
        </div>
      </div>

      <Tabs defaultValue="env">
        <TabsList>
          <TabsTrigger value="env">.env</TabsTrigger>
          <TabsTrigger value="mem0">.env.mem0</TabsTrigger>
        </TabsList>
        <TabsContent value="env">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Variables de entorno</CardTitle>
              <CardDescription>HOST_*, MCP_TOOLS_*, MEM0_USER_ID.</CardDescription>
            </CardHeader>
            <CardContent>
              <EnvTable
                title=".env"
                keys={ENV_KEYS}
                data={data?.env}
                onSubmit={async (values) => {
                  const res = await api("/api/env", { method: "POST", body: { values } });
                  qc.invalidateQueries({ queryKey: ["status"] });
                  return res;
                }}
              />
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="mem0">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">mem0</CardTitle>
              <CardDescription>Proveedor, modelos, paths.</CardDescription>
            </CardHeader>
            <CardContent>
              <EnvTable
                title=".env.mem0"
                keys={MEM0_KEYS}
                data={data?.env_mem0}
                onSubmit={async (values) => {
                  const res = await api("/api/env-mem0", { method: "POST", body: { values } });
                  qc.invalidateQueries({ queryKey: ["status"] });
                  return res;
                }}
              />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Separator />

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <SettingsIcon className="h-4 w-4" />
            Sincronizadores
          </CardTitle>
          <CardDescription>
            Re-registra MCPs, skills o reglas en los clientes soportados.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            disabled={syncMut.isPending}
            onClick={() => syncMut.mutate("/api/mcp-config/sync")}
          >
            <RefreshCcw className="h-4 w-4" /> Re-run mcp-config
          </Button>
          <Button
            variant="outline"
            disabled={syncMut.isPending}
            onClick={() => syncMut.mutate("/api/skills/sync")}
          >
            <RefreshCcw className="h-4 w-4" /> Sync skills
          </Button>
          <Button
            variant="outline"
            disabled={syncMut.isPending}
            onClick={() => syncMut.mutate("/api/rules/sync")}
          >
            <RefreshCcw className="h-4 w-4" /> Sync rules
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

// keep Pair type re-exported for tree-shake friendliness
export type { Pair };