import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, Terminal, Trash2 } from "lucide-react";
import { useEventSource, type JobLine } from "~/lib/sse";
import { api, type ServiceView } from "~/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Switch } from "~/components/ui/switch";
import { Label } from "~/components/ui/label";
import { Input } from "~/components/ui/input";

export default function LogsRoute() {
  const { data: services } = useQuery<ServiceView[]>({
    queryKey: ["services"],
    queryFn: () => api<ServiceView[]>("/api/services"),
    refetchInterval: 5_000,
  });
  const [service, setService] = useState<string>("");
  const [follow, setFollow] = useState(true);
  const [tail, setTail] = useState(200);
  const [lines, setLines] = useState<JobLine[]>([]);
  const preRef = useRef<HTMLPreElement>(null);
  useEffect(() => {
    setLines([]);
  }, [service]);

  const url = service
    ? `/api/logs/${encodeURIComponent(service)}?tail=${tail}&follow=${follow ? 1 : 0}`
    : null;

  useEventSource(url, (evt) => {
    setLines((prev) => {
      const next = [...prev, evt];
      const trimmed = next.length > 2000 ? next.slice(next.length - 2000) : next;
      queueMicrotask(() => {
        if (preRef.current) {
          preRef.current.scrollTop = preRef.current.scrollHeight;
        }
      });
      return trimmed;
    });
  });

  function download() {
    const blob = new Blob([lines.map((l) => l.text).join("\n")], { type: "text/plain" });
    const u = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = u;
    a.download = `${service || "service"}.log`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(u);
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Terminal className="h-4 w-4" />
            Stream
          </CardTitle>
          <CardDescription>Selecciona un servicio y (opcional) síguelo.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="svc">Servicio</Label>
              <select
                id="svc"
                value={service}
                onChange={(e) => setService(e.target.value)}
                className="h-9 rounded-md border border-border bg-background px-2 text-sm"
              >
                <option value="">— elige —</option>
                {services?.map((s) => (
                  <option key={s.name} value={s.name}>
                    {s.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="tail">tail</Label>
              <Input
                id="tail"
                type="number"
                min={10}
                max={5000}
                value={tail}
                onChange={(e) => setTail(Number(e.target.value) || 200)}
                className="w-24"
              />
            </div>
            <div className="flex items-center gap-2 pb-1">
              <Switch id="follow" checked={follow} onCheckedChange={setFollow} />
              <Label htmlFor="follow">follow</Label>
            </div>
            <div className="ml-auto flex gap-2">
              <Button variant="outline" onClick={() => setLines([])}>
                <Trash2 className="h-4 w-4" /> Clear
              </Button>
              <Button variant="outline" onClick={download} disabled={lines.length === 0}>
                <Download className="h-4 w-4" /> Descargar
              </Button>
            </div>
          </div>
          <pre
            ref={preRef}
            className="max-h-[60vh] overflow-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs whitespace-pre-wrap"
          >
            {lines.length === 0
              ? service
                ? "Esperando salida…"
                : "Selecciona un servicio para empezar."
              : lines.map((l, i) => (
                  <div
                    key={i}
                    className={l.stream === "stderr" ? "text-warning" : "text-foreground/90"}
                  >
                    {l.text}
                  </div>
                ))}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}