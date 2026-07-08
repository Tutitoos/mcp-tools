import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion, useMotionValue, useTransform, animate } from "motion/react";
import { Activity, Box, Container, Sparkles } from "lucide-react";
import { api, type StatusPayload } from "~/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "~/components/ui/card";
import { Badge } from "~/components/ui/badge";
import { Skeleton } from "~/components/ui/skeleton";

function CountUp({ value, suffix = "" }: { value: number; suffix?: string }) {
  const mv = useMotionValue(0);
  const display = useTransform(mv, (latest) => Math.round(latest).toString());
  const [text, setText] = useState("0");
  useEffect(() => {
    const controls = animate(mv, value, { duration: 0.8, ease: "easeOut" });
    const unsub = display.on("change", (v) => setText(v));
    return () => {
      controls.stop();
      unsub();
    };
  }, [value, mv, display]);
  return (
    <span className="font-mono tabular-nums">
      {text}
      {suffix}
    </span>
  );
}

function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  accent,
}: {
  label: string;
  value: number;
  icon: React.ComponentType<{ className?: string }>;
  hint: string;
  accent: string;
}) {
  return (
    <Card className="relative overflow-hidden">
      <div
        className="pointer-events-none absolute inset-0 opacity-30"
        style={{
          background: `radial-gradient(60% 80% at 0% 0%, ${accent} 0%, transparent 60%)`,
        }}
      />
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-semibold">
          <CountUp value={value} />
        </div>
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}

function SkeletonStat() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <Skeleton className="h-3 w-20" />
        <Skeleton className="h-4 w-4 rounded" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-8 w-16" />
        <Skeleton className="mt-2 h-3 w-32" />
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  const { data, isLoading } = useQuery<StatusPayload>({
    queryKey: ["status"],
    queryFn: () => api<StatusPayload>("/api/status"),
    refetchInterval: 5_000,
  });

  const installed = data
    ? data.compose_services.filter((s) => s.state === "running").length
    : 0;
  const registry = 16;
  const selected = data?.state.selected.length ?? 0;
  const services = data?.compose_services.length ?? 0;

  return (
    <div className="space-y-8">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: "easeOut" }}
        className="space-y-2"
      >
        <Badge variant="outline" className="gap-2">
          <Sparkles className="h-3 w-3" />
          Panel de control
        </Badge>
        <h1 className="text-3xl font-semibold tracking-tight">
          Hola{data?.env_mem0?.MEM0_USER_ID ? `, ${data.env_mem0.MEM0_USER_ID}` : ""}
        </h1>
        <p className="text-muted-foreground">
          Tu stack MCP en un vistazo. Las cifras se actualizan cada 5 segundos.
        </p>
      </motion.div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {isLoading || !data ? (
          <>
            <SkeletonStat />
            <SkeletonStat />
            <SkeletonStat />
            <SkeletonStat />
          </>
        ) : (
          <>
            <StatCard
              label="Tools seleccionadas"
              value={selected}
              icon={Box}
              hint="Componentes activos del state.json"
              accent="#a855f7"
            />
            <StatCard
              label="Servicios corriendo"
              value={installed}
              icon={Container}
              hint={`${services} servicios docker totales`}
              accent="#38bdf8"
            />
            <StatCard
              label="En el registro"
              value={registry}
              icon={Activity}
              hint="Componentes disponibles para instalar"
              accent="#ec4899"
            />
            <StatCard
              label="Última actualización"
              value={0}
              icon={Sparkles}
              hint={
                data.state.updated_at
                  ? new Date(data.state.updated_at).toLocaleString("es-ES")
                  : "—"
              }
              accent="#fbbf24"
            />
          </>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Servicios docker</CardTitle>
          <CardDescription>Estado en vivo de los contenedores definidos en dockers/compose.yaml.</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading || !data ? (
            <div className="grid gap-2 sm:grid-cols-2">
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : data.compose_services.length === 0 ? (
            <p className="text-sm text-muted-foreground">Aún no hay servicios docker.</p>
          ) : (
            <div className="grid gap-2 sm:grid-cols-2">
              {data.compose_services.map((svc) => (
                <div
                  key={svc.name}
                  className="flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2"
                >
                  <span className="font-mono text-sm">{svc.name}</span>
                  <Badge variant={svc.state === "running" ? "success" : "secondary"}>
                    {svc.state}
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}