import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useLocation, useMatches } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { motion, MotionConfig } from "motion/react";
import {
  Activity,
  Boxes,
  Cog,
  Database,
  Gauge,
  Layers,
  Menu,
  Moon,
  Puzzle,
  Settings as SettingsIcon,
  Sun,
  TerminalSquare,
} from "lucide-react";
import { useTheme } from "next-themes";
import { api, type StatusPayload, type VersionResponse } from "~/lib/api";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetTrigger,
  SheetClose,
} from "~/components/ui/sheet";
import { cn } from "~/lib/utils";

type NavItem = {
  to: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
};

const NAV_GENERAL: NavItem[] = [
  { to: "/", label: "Dashboard", icon: Gauge },
  { to: "/tools", label: "Tools", icon: Boxes },
  { to: "/plugins", label: "Plugins", icon: Puzzle },
  { to: "/jobs", label: "Jobs", icon: Activity },
  { to: "/models", label: "Modelos", icon: Database },
  { to: "/logs", label: "Logs", icon: TerminalSquare },
];

const NAV_ADMIN: NavItem[] = [
  { to: "/configure", label: "Configurar", icon: Layers },
  { to: "/services", label: "Servicios", icon: Cog },
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
      aria-hidden="true"
    >
      <rect
        x="3"
        y="3"
        width="26"
        height="26"
        rx="7"
        className="fill-primary/15 stroke-primary"
        strokeWidth="1.5"
      />
      <path
        d="M10 11h12M10 16h8M10 21h12"
        className="stroke-foreground/80"
        strokeWidth="1.6"
        strokeLinecap="round"
        fill="none"
      />
    </motion.svg>
  );
}

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const current = mounted ? resolvedTheme : "dark";
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme(current === "dark" ? "light" : "dark")}
      aria-label="Cambiar tema"
      suppressHydrationWarning
    >
      {current === "dark" ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
    </Button>
  );
}

function UnitStatusPill() {
  const { data, isLoading, error } = useQuery<StatusPayload>({
    queryKey: ["status"],
    queryFn: () => api<StatusPayload>("/api/status"),
    refetchInterval: 5_000,
  });
  if (error) {
    return (
      <Badge
        variant="destructive"
        className="gap-2 px-3 py-1"
        title={(error as Error).message}
      >
        <span className="h-1.5 w-1.5 rounded-full bg-current" />
        <span className="hidden sm:inline">estado: error</span>
      </Badge>
    );
  }
  if (isLoading || !data) {
    return (
      <Badge variant="outline" className="gap-2 px-3 py-1">
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-muted-foreground" />
        <span className="hidden sm:inline">Cargando…</span>
      </Badge>
    );
  }
  const services = data.compose_services ?? [];
  const running = services.filter((s) => s.state === "running").length;
  const total = services.length;
  const variant =
    data.docker_running && running > 0
      ? "success"
      : data.docker_running
        ? "warning"
        : "destructive";
  const shortLabel = data.docker_running
    ? `docker · ${running}/${total}`
    : "docker inactivo";
  return (
    <Badge
      variant={variant}
      className="max-w-[8rem] truncate gap-2 px-3 py-1 lg:max-w-none"
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      <span className="lg:hidden">{shortLabel}</span>
      <span className="hidden lg:inline">
        {data.docker_running
          ? `docker · ${running}/${total} servicios`
          : "docker inactivo"}
      </span>
    </Badge>
  );
}

function NavRow({ to, label, icon: Icon }: NavItem) {
  const location = useLocation();
  const active =
    to === "/" ? location.pathname === "/" : location.pathname.startsWith(to);
  return (
    <NavLink
      to={to}
      className={cn(
        "mx-2 flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
        active
          ? "bg-accent text-foreground"
          : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
      )}
    >
      <Icon className="h-4 w-4" />
      {label}
    </NavLink>
  );
}

function SidebarBrand() {
  const { data } = useQuery<VersionResponse>({
    queryKey: ["version"],
    queryFn: () => api<VersionResponse>("/api/version"),
  });
  return (
    <Link
      to="/"
      className="flex items-center gap-2.5 overflow-hidden border-b border-border px-4 py-4"
    >
      <LogoMark />
      <span className="shrink-0 whitespace-nowrap font-mono text-sm font-semibold tracking-tight">
        mcp-tools
      </span>
      <span
        className="truncate text-[10px] uppercase text-muted-foreground"
        title={data?.version}
      >
        {data
          ? data.version.startsWith("v")
            ? data.version
            : `v${data.version}`
          : "—"}
      </span>
    </Link>
  );
}

function NavSection({ title, items }: { title: string; items: NavItem[] }) {
  return (
    <div>
      <div className="px-3 py-2 text-[11px] uppercase tracking-widest text-muted-foreground">
        {title}
      </div>
      <nav className="flex flex-col gap-1">
        {items.map((item) => (
          <NavRow key={item.to} {...item} />
        ))}
      </nav>
    </div>
  );
}

function Sidebar() {
  return (
    <aside className="hidden md:flex md:flex-col border-r border-border bg-[hsl(var(--card))]">
      <SidebarBrand />
      <NavSection title="GENERAL" items={NAV_GENERAL} />
      <NavSection title="ADMINISTRACIÓN" items={NAV_ADMIN} />
      <div className="flex-1" />
      <div className="border-t border-border px-4 py-3">
        <UnitStatusPill />
      </div>
    </aside>
  );
}

function MobileNav() {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Abrir menú"
          data-mobile-nav-trigger
        >
          <Menu className="h-4 w-4" />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="flex flex-col gap-2">
        <Link to="/" className="flex items-center gap-2.5 px-2 pb-2">
          <LogoMark />
          <span className="font-mono text-sm font-semibold tracking-tight">
            mcp-tools
          </span>
        </Link>
        <div>
          <div className="px-1 py-2 text-[11px] uppercase tracking-widest text-muted-foreground">
            GENERAL
          </div>
          <nav className="flex flex-col gap-1">
            {NAV_GENERAL.map((item) => (
              <SheetClose asChild key={item.to}>
                <NavRow {...item} />
              </SheetClose>
            ))}
          </nav>
          <div className="px-1 py-2 text-[11px] uppercase tracking-widest text-muted-foreground">
            ADMINISTRACIÓN
          </div>
          <nav className="flex flex-col gap-1">
            {NAV_ADMIN.map((item) => (
              <SheetClose asChild key={item.to}>
                <NavRow {...item} />
              </SheetClose>
            ))}
          </nav>
        </div>
      </SheetContent>
    </Sheet>
  );
}

function TopBar() {
  const matches = useMatches();
  const title =
    (matches.at(-1)?.handle as { title?: string } | undefined)?.title ?? "";
  return (
    <header className="glass sticky top-0 z-40 flex h-14 items-center gap-3 border-b border-border px-4 lg:px-6">
      <div className="md:hidden">
        <MobileNav />
      </div>
      <h1 className="invisible flex-1 text-sm font-medium md:visible">
        {title}
      </h1>
      <ThemeToggle />
    </header>
  );
}

export default function Shell() {
  return (
    <MotionConfig reducedMotion="user">
      <div className="grid min-h-screen w-full md:grid-cols-[240px_1fr]">
        <Sidebar />
        <div className="flex min-w-0 flex-col">
          <TopBar />
          <main className="flex-1 px-6 py-6 lg:px-8 lg:py-8">
            <Outlet />
          </main>
        </div>
      </div>
    </MotionConfig>
  );
}
