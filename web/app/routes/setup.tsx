import { useState } from "react";
import { useNavigate } from "react-router";
import { motion } from "motion/react";
import { KeyRound } from "lucide-react";
import { setToken } from "~/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Label } from "~/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";

export default function SetupRoute() {
  const nav = useNavigate();
  const [token, setLocal] = useState("");
  const [err, setErr] = useState<string | null>(null);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const t = token.trim();
    if (t.length < 8) {
      setErr("El token parece demasiado corto. Cópialo del output de `mcp-tools install`.");
      return;
    }
    setToken(t);
    nav("/", { replace: true });
  }

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: "easeOut" }}
        className="w-full max-w-md"
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="h-5 w-5" />
              Configuración inicial
            </CardTitle>
            <CardDescription>
              Pega el token que imprimió <code>mcp-tools install</code>. Se guarda en
              este navegador (localStorage).
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form className="space-y-3" onSubmit={submit}>
              <div className="grid gap-1">
                <Label htmlFor="token">Bearer token</Label>
                <Input
                  id="token"
                  autoFocus
                  placeholder="abcdef0123…"
                  value={token}
                  onChange={(e) => setLocal(e.target.value)}
                />
              </div>
              {err && (
                <Alert variant="destructive">
                  <AlertTitle>Token inválido</AlertTitle>
                  <AlertDescription>{err}</AlertDescription>
                </Alert>
              )}
              <Button type="submit" className="w-full">
                Guardar y entrar
              </Button>
            </form>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}