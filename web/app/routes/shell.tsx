import { Link, NavLink, Outlet, useLocation } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { motion, MotionConfig } from "motion/react";
import {
  Boxes,
  Cog,
  Database,
  FileText,
  Gauge,
  Layers,
  Moon,
  Settings as SettingsIcon,
  Sun,
  TerminalSquare,
} from "lucide-react";
import { useTheme } from "next-themes";
import { api, type StatusPayload } from "~/lib/api";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { Separator } from "~/components/ui/separator";
import { cn } from "~/lib/utils";

type NavItem = {
  to: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
};

const NAV: NavItem[] = [
  { to: "/", label: "Dashboard", icon: Gauge },
  { to: "/tools", label: "Tools", icon: Boxes },
  { to: "/configure", label: "Configurar", icon: Layers },
  { to: "/models", label: "Modelos", icon: Database },
  { to: "/services", label: "Servicios", icon: Cog },
  { to: "/logs", label: "Logs", icon: TerminalSquare },
  { to: "/settings", label: "Settings", icon: SettingsIcon },
];

function LogoMark() {
  return (
    <motion.svg
      viewBox="0 0 32 32"
      className="h-6 w-6"
      initial={{ rotate: -8 }}
      animate={{ rotate: 0 }}
      transition={{ type: "spring", stiffness: 200, damping: 14 }}
    >
      <defs>
        <linearGradient id="lg" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#a855f7" />
          <stop offset="50%" stopColor="#ec4899" />
          <stop offset="100%" stopColor="#38bdf8" />
        </linearGradient>
      </defs>
      <rect x="3" y="3" width="26" height="26" rx="7" fill="url(#lg)" opacity="0.18" />
      <rect x="3" y="3" width="26" height="26" rx="7" stroke="url(#lg)" strokeWidth="1.5" fill="none" />
      <path
        d="M10 11h12M10 16h8M10 21h12"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        opacity="0.85"
      />
    </motion.svg>
  );
}

function ThemeToggle() {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const current = theme === "system" ? resolvedTheme : theme;
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme(current === "dark" ? "light" : "dark")}
      aria-label="Cambiar tema"
    >
      {current === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </Button>
  );
}

function UnitStatusPill() {
  const { data, isLoading } = useQuery<StatusPayload>({
    queryKey: ["status"],
    queryFn: () => api<StatusPayload>("/api/status"),
    refetchInterval: 5_000,
  });
  if (isLoading || !data) {
    return (
      <Badge variant="outline" className="gap-2 px-3 py-1">
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-muted-foreground" />
        Cargando…
      </Badge>
    );
  }
  const running = data.compose_services.filter((s) => s.state === "running").length;
  const total = data.compose_services.length;
  const variant =
    data.docker_running && running > 0
      ? "success"
      : data.docker_running
        ? "warning"
        : "destructive";
  return (
    <Badge variant={variant} className="gap-2 px-3 py-1">
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {data.docker_running
        ? `docker · ${running}/${total} servicios`
        : "docker inactivo"}
    </Badge>
  );
}

export default function Shell() {
  const location = useLocation();
  return (
    <MotionConfig reducedMotion="user">
      <div className="flex min-h-screen flex-col">
        <header className="glass sticky top-0 z-40 border-b border-border">
          <div className="mx-auto flex h-14 max-w-7xl items-center gap-6 px-4 lg:px-6">
            <Link to="/" className="flex items-center gap-2.5">
              <LogoMark />
              <span className="font-mono text-sm font-semibold tracking-tight">
                mcp-tools
              </span>
            </Link>
            <nav className="hidden flex-1 items-center gap-1 md:flex">
              {NAV.map((item) => {
                const active =
                  item.to === "/"
                    ? location.pathname === "/"
                    : location.pathname.startsWith(item.to);
                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={cn(
                      "group inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors",
                      active
                        ? "bg-accent text-foreground"
                        : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
                    )}
                  >
                    <item.icon className="h-3.5 w-3.5" />
                    {item.label}
                  </NavLink>
                );
              })}
            </nav>
            <div className="ml-auto flex items-center gap-2">
              <UnitStatusPill />
              <ThemeToggle />
            </div>
          </div>
        </header>
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-8 lg:px-6">
          <Outlet />
        </main>
        <footer className="border-t border-border/60 py-4 text-center text-xs text-muted-foreground">
          mcp-tools · web admin panel
        </footer>
        <Separator className="hidden" />
      </div>
    </MotionConfig>
  );
}

// Re-export FileText so the icon import isn't tree-shaken in dev previews.
export { FileText };