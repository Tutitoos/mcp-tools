import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion, useMotionValue, useTransform, animate } from "motion/react";
import { Activity, Box, Container, Sparkles } from "lucide-react";
import { api, type StatusPayload, type ToolView } from "~/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "~/components/ui/card";
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
}: {
  label: string;
  value: number;
  icon: React.ComponentType<{ className?: string }>;
  hint: string;
}) {
  return (
    <Card>
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

  // Shared cache key with /tools and /configure — first visit to either
  // route warms the cache; this component reuses it without an extra
  // network request.
  const { data: tools } = useQuery<ToolView[]>({
    queryKey: ["tools"],
    queryFn: () => api<ToolView[]>("/api/tools"),
    refetchInterval: 5_000,
  });

  const compose = data?.compose_services ?? [];
  const installed = compose.filter((s) => s.state === "running").length;
  const registry = tools?.length ?? 0;
  const selected = data?.state?.selected?.length ?? 0;
  let updatedAt = "—";
  if (data?.state?.updated_at) {
    const d = new Date(data.state.updated_at);
    if (!isNaN(d.getTime())) updatedAt = d.toLocaleString("es-ES");
  }

  return (
    <div className="space-y-8">
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
            />
            <StatCard
              label="Servicios corriendo"
              value={installed}
              icon={Container}
              hint={`${compose.length} servicios docker totales`}
            />
            <StatCard
              label="En el registro"
              value={registry}
              icon={Activity}
              hint="Componentes disponibles para instalar"
            />
            <Card className="relative overflow-hidden">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Última actualización
                </CardTitle>
                <Sparkles className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <p className="font-mono text-lg tracking-tight">{updatedAt}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Última escritura en state.json
                </p>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </div>
  );
}
