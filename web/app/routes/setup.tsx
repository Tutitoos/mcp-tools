import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { motion } from "motion/react";
import { KeyRound, LogIn } from "lucide-react";
import { getToken, setToken } from "~/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "~/components/ui/card";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Label } from "~/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";

export default function SetupRoute() {
  const nav = useNavigate();
  const [token, setLocal] = useState(getToken() ?? "");
  const [err, setErr] = useState<string | null>(null);
  const [required, setRequired] = useState(!getToken());

  // If the API client fires mcp-tools:unauthorized while the user is on
  // any other route, route them here so they can re-authenticate.
  useEffect(() => {
    function onUnauth() {
      setRequired(true);
    }
    window.addEventListener("mcp-tools:unauthorized", onUnauth);
    return () => window.removeEventListener("mcp-tools:unauthorized", onUnauth);
  }, []);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const t = token.trim();
    if (t.length < 8) {
      setErr("El token parece demasiado corto. Cópialo del output de `mcp-tools install`.");
      return;
    }
    setToken(t);
    setRequired(false);
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
              {required ? <LogIn className="h-5 w-5" /> : <KeyRound className="h-5 w-5" />}
              {required ? "Iniciar sesión" : "Configuración inicial"}
            </CardTitle>
            <CardDescription>
              {required
                ? "El panel requiere autenticación. Pega el token para continuar."
                : "Pega el token que imprimió "}
              {!required && <code>mcp-tools install</code>}
              {!required && ". Se guarda en este navegador (localStorage)."}
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
                {required ? "Entrar" : "Guardar y entrar"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}