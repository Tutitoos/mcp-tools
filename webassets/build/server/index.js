var __defProp = Object.defineProperty;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);
import { jsx, jsxs, Fragment } from "react/jsx-runtime";
import { PassThrough } from "node:stream";
import { ServerRouter, UNSAFE_withComponentProps, Outlet, UNSAFE_withErrorBoundaryProps, useRouteError, isRouteErrorResponse, Meta, Links, ScrollRestoration, Scripts, useLocation, Link, NavLink, useNavigate } from "react-router";
import { createReadableStreamFromReadable } from "@react-router/node";
import { isbot } from "isbot";
import { renderToPipeableStream } from "react-dom/server";
import { ThemeProvider, useTheme } from "next-themes";
import { Toaster, toast } from "sonner";
import { QueryClient, QueryClientProvider, useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import * as React from "react";
import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { MotionConfig, motion, useMotionValue, useTransform, animate, AnimatePresence } from "motion/react";
import { Gauge, Boxes, Layers, Database, Cog, TerminalSquare, Settings, Sun, Moon, FileText, Sparkles, Box, Container, Activity, X, AlertCircle, CheckCircle2, XCircle, Loader2, Download, RefreshCcw, Trash2, Save, Play, Square, RotateCcw, ScrollText, Terminal, KeyRound } from "lucide-react";
import { Slot } from "@radix-ui/react-slot";
import { cva } from "class-variance-authority";
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import * as SeparatorPrimitive from "@radix-ui/react-separator";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import * as LabelPrimitive from "@radix-ui/react-label";
import * as TabsPrimitive from "@radix-ui/react-tabs";
const streamTimeout = 5e3;
function handleRequest(request, responseStatusCode, responseHeaders, routerContext) {
  return new Promise((resolve, reject) => {
    let shellRendered = false;
    const userAgent = request.headers.get("user-agent");
    const isBot = userAgent ? isbot(userAgent) : false;
    const readyEvent = isBot ? "onAllReady" : "onShellReady";
    const { pipe, abort } = renderToPipeableStream(
      /* @__PURE__ */ jsx(ServerRouter, { context: routerContext, url: request.url }),
      {
        [readyEvent]() {
          shellRendered = true;
          const body = new PassThrough();
          const stream = createReadableStreamFromReadable(body);
          responseHeaders.set("Content-Type", "text/html");
          resolve(
            new Response(stream, {
              headers: responseHeaders,
              status: responseStatusCode
            })
          );
          pipe(body);
        },
        onShellError(error) {
          reject(error);
        },
        onError(error) {
          responseStatusCode = 500;
          if (shellRendered) {
            console.error(error);
          }
        }
      }
    );
    setTimeout(abort, streamTimeout + 1e3);
  });
}
const entryServer = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: handleRequest,
  streamTimeout
}, Symbol.toStringTag, { value: "Module" }));
const preconnectGoogle = {
  rel: "preconnect",
  href: "https://fonts.googleapis.com"
};
const preconnectGstatic = {
  rel: "preconnect",
  href: "https://fonts.gstatic.com",
  crossOrigin: "anonymous"
};
const geistStylesheet = {
  rel: "stylesheet",
  href: "https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500;600&display=swap"
};
const links = () => [preconnectGoogle, preconnectGstatic, geistStylesheet];
function Layout({
  children
}) {
  return /* @__PURE__ */ jsxs("html", {
    lang: "es",
    suppressHydrationWarning: true,
    children: [/* @__PURE__ */ jsxs("head", {
      children: [/* @__PURE__ */ jsx("meta", {
        charSet: "utf-8"
      }), /* @__PURE__ */ jsx("meta", {
        name: "viewport",
        content: "width=device-width, initial-scale=1"
      }), /* @__PURE__ */ jsx(Meta, {}), /* @__PURE__ */ jsx(Links, {})]
    }), /* @__PURE__ */ jsxs("body", {
      className: "min-h-screen bg-background font-sans antialiased",
      children: [/* @__PURE__ */ jsx(ThemeProvider, {
        attribute: "class",
        defaultTheme: "dark",
        enableSystem: true,
        disableTransitionOnChange: false,
        children: /* @__PURE__ */ jsxs("div", {
          className: "relative isolate",
          children: [/* @__PURE__ */ jsx("div", {
            className: "gradient-mesh pointer-events-none fixed inset-0 -z-10"
          }), children, /* @__PURE__ */ jsx(Toaster, {
            richColors: true,
            position: "top-right"
          })]
        })
      }), /* @__PURE__ */ jsx(ScrollRestoration, {}), /* @__PURE__ */ jsx(Scripts, {})]
    })]
  });
}
const root = UNSAFE_withComponentProps(function App() {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 2e3,
        refetchOnWindowFocus: false
      }
    }
  }));
  return /* @__PURE__ */ jsx(QueryClientProvider, {
    client: queryClient,
    children: /* @__PURE__ */ jsx(Outlet, {})
  });
});
const ErrorBoundary = UNSAFE_withErrorBoundaryProps(function ErrorBoundary2() {
  const error = useRouteError();
  let title = "Algo salió mal";
  let detail = "Error inesperado.";
  if (isRouteErrorResponse(error)) {
    title = `${error.status} ${error.statusText}`;
    detail = typeof error.data === "string" ? error.data : detail;
  } else if (error instanceof Error) {
    detail = error.message;
  }
  return /* @__PURE__ */ jsx("div", {
    className: "flex min-h-screen items-center justify-center p-6",
    children: /* @__PURE__ */ jsxs("div", {
      className: "card-vc max-w-lg p-6",
      children: [/* @__PURE__ */ jsx("h1", {
        className: "text-xl font-semibold",
        children: title
      }), /* @__PURE__ */ jsx("p", {
        className: "mt-2 text-sm text-muted-foreground",
        children: detail
      })]
    })
  });
});
const route0 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  ErrorBoundary,
  Layout,
  default: root,
  links
}, Symbol.toStringTag, { value: "Module" }));
const TOKEN_KEY = "mcp-tools-token";
function getToken() {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}
function setToken(token) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
}
class ApiError extends Error {
  constructor(status, message, body) {
    super(message);
    __publicField(this, "status");
    __publicField(this, "body");
    this.status = status;
    this.body = body;
  }
}
async function api(path, init = {}) {
  const headers = new Headers(init.headers ?? {});
  headers.set("Accept", "application/json");
  let body;
  if (init.body !== void 0) {
    if (init.body instanceof FormData) {
      body = init.body;
    } else {
      headers.set("Content-Type", "application/json");
      body = JSON.stringify(init.body);
    }
  }
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(path, { ...init, headers, body });
  const text = await res.text();
  let parsed = text;
  if (text.length > 0) {
    try {
      parsed = JSON.parse(text);
    } catch {
    }
  }
  if (!res.ok) {
    const message = parsed && typeof parsed === "object" && "error" in parsed ? String(parsed.error) : res.statusText;
    throw new ApiError(res.status, message, parsed);
  }
  return parsed;
}
async function apiStream(path) {
  const headers = new Headers();
  headers.set("Accept", "text/event-stream");
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetch(path, { headers });
}
function cn(...inputs) {
  return twMerge(clsx(inputs));
}
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90 shadow",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline: "border border-border bg-transparent hover:bg-accent hover:text-accent-foreground",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline"
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-6",
        icon: "h-9 w-9"
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default"
    }
  }
);
const Button = React.forwardRef(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return /* @__PURE__ */ jsx(
      Comp,
      {
        className: cn(buttonVariants({ variant, size, className })),
        ref,
        ...props
      }
    );
  }
);
Button.displayName = "Button";
const badgeVariants = cva(
  "inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium transition-colors focus:outline-none",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary: "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        destructive: "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        outline: "text-foreground border-border",
        success: "border-transparent bg-emerald-500/15 text-emerald-300",
        warning: "border-transparent bg-amber-500/15 text-amber-300"
      }
    },
    defaultVariants: { variant: "default" }
  }
);
function Badge({ className, variant, ...props }) {
  return /* @__PURE__ */ jsx("div", { className: cn(badgeVariants({ variant }), className), ...props });
}
const Separator = React.forwardRef(
  ({ className, orientation = "horizontal", decorative = true, ...props }, ref) => /* @__PURE__ */ jsx(
    SeparatorPrimitive.Root,
    {
      ref,
      decorative,
      orientation,
      className: cn(
        "shrink-0 bg-border",
        orientation === "horizontal" ? "h-px w-full" : "h-full w-px",
        className
      ),
      ...props
    }
  )
);
Separator.displayName = SeparatorPrimitive.Root.displayName;
const NAV = [{
  to: "/",
  label: "Dashboard",
  icon: Gauge
}, {
  to: "/tools",
  label: "Tools",
  icon: Boxes
}, {
  to: "/configure",
  label: "Configurar",
  icon: Layers
}, {
  to: "/models",
  label: "Modelos",
  icon: Database
}, {
  to: "/services",
  label: "Servicios",
  icon: Cog
}, {
  to: "/logs",
  label: "Logs",
  icon: TerminalSquare
}, {
  to: "/settings",
  label: "Settings",
  icon: Settings
}];
function LogoMark() {
  return /* @__PURE__ */ jsxs(motion.svg, {
    viewBox: "0 0 32 32",
    className: "h-6 w-6",
    initial: {
      rotate: -8
    },
    animate: {
      rotate: 0
    },
    transition: {
      type: "spring",
      stiffness: 200,
      damping: 14
    },
    children: [/* @__PURE__ */ jsx("defs", {
      children: /* @__PURE__ */ jsxs("linearGradient", {
        id: "lg",
        x1: "0",
        y1: "0",
        x2: "1",
        y2: "1",
        children: [/* @__PURE__ */ jsx("stop", {
          offset: "0%",
          stopColor: "#a855f7"
        }), /* @__PURE__ */ jsx("stop", {
          offset: "50%",
          stopColor: "#ec4899"
        }), /* @__PURE__ */ jsx("stop", {
          offset: "100%",
          stopColor: "#38bdf8"
        })]
      })
    }), /* @__PURE__ */ jsx("rect", {
      x: "3",
      y: "3",
      width: "26",
      height: "26",
      rx: "7",
      fill: "url(#lg)",
      opacity: "0.18"
    }), /* @__PURE__ */ jsx("rect", {
      x: "3",
      y: "3",
      width: "26",
      height: "26",
      rx: "7",
      stroke: "url(#lg)",
      strokeWidth: "1.5",
      fill: "none"
    }), /* @__PURE__ */ jsx("path", {
      d: "M10 11h12M10 16h8M10 21h12",
      stroke: "currentColor",
      strokeWidth: "1.6",
      strokeLinecap: "round",
      opacity: "0.85"
    })]
  });
}
function ThemeToggle() {
  const {
    theme,
    setTheme,
    resolvedTheme
  } = useTheme();
  const current = theme === "system" ? resolvedTheme : theme;
  return /* @__PURE__ */ jsx(Button, {
    variant: "ghost",
    size: "icon",
    onClick: () => setTheme(current === "dark" ? "light" : "dark"),
    "aria-label": "Cambiar tema",
    children: current === "dark" ? /* @__PURE__ */ jsx(Sun, {
      className: "h-4 w-4"
    }) : /* @__PURE__ */ jsx(Moon, {
      className: "h-4 w-4"
    })
  });
}
function UnitStatusPill() {
  const {
    data,
    isLoading
  } = useQuery({
    queryKey: ["status"],
    queryFn: () => api("/api/status"),
    refetchInterval: 5e3
  });
  if (isLoading || !data) {
    return /* @__PURE__ */ jsxs(Badge, {
      variant: "outline",
      className: "gap-2 px-3 py-1",
      children: [/* @__PURE__ */ jsx("span", {
        className: "h-1.5 w-1.5 animate-pulse rounded-full bg-muted-foreground"
      }), "Cargando…"]
    });
  }
  const running = data.compose_services.filter((s) => s.state === "running").length;
  const total = data.compose_services.length;
  const variant = data.docker_running && running > 0 ? "success" : data.docker_running ? "warning" : "destructive";
  return /* @__PURE__ */ jsxs(Badge, {
    variant,
    className: "gap-2 px-3 py-1",
    children: [/* @__PURE__ */ jsx("span", {
      className: "h-1.5 w-1.5 rounded-full bg-current"
    }), data.docker_running ? `docker · ${running}/${total} servicios` : "docker inactivo"]
  });
}
const shell = UNSAFE_withComponentProps(function Shell() {
  const location = useLocation();
  return /* @__PURE__ */ jsx(MotionConfig, {
    reducedMotion: "user",
    children: /* @__PURE__ */ jsxs("div", {
      className: "flex min-h-screen flex-col",
      children: [/* @__PURE__ */ jsx("header", {
        className: "glass sticky top-0 z-40 border-b border-border",
        children: /* @__PURE__ */ jsxs("div", {
          className: "mx-auto flex h-14 max-w-7xl items-center gap-6 px-4 lg:px-6",
          children: [/* @__PURE__ */ jsxs(Link, {
            to: "/",
            className: "flex items-center gap-2.5",
            children: [/* @__PURE__ */ jsx(LogoMark, {}), /* @__PURE__ */ jsx("span", {
              className: "font-mono text-sm font-semibold tracking-tight",
              children: "mcp-tools"
            })]
          }), /* @__PURE__ */ jsx("nav", {
            className: "hidden flex-1 items-center gap-1 md:flex",
            children: NAV.map((item) => {
              const active = item.to === "/" ? location.pathname === "/" : location.pathname.startsWith(item.to);
              return /* @__PURE__ */ jsxs(NavLink, {
                to: item.to,
                className: cn("group inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors", active ? "bg-accent text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"),
                children: [/* @__PURE__ */ jsx(item.icon, {
                  className: "h-3.5 w-3.5"
                }), item.label]
              }, item.to);
            })
          }), /* @__PURE__ */ jsxs("div", {
            className: "ml-auto flex items-center gap-2",
            children: [/* @__PURE__ */ jsx(UnitStatusPill, {}), /* @__PURE__ */ jsx(ThemeToggle, {})]
          })]
        })
      }), /* @__PURE__ */ jsx("main", {
        className: "mx-auto w-full max-w-7xl flex-1 px-4 py-8 lg:px-6",
        children: /* @__PURE__ */ jsx(Outlet, {})
      }), /* @__PURE__ */ jsx("footer", {
        className: "border-t border-border/60 py-4 text-center text-xs text-muted-foreground",
        children: "mcp-tools · web admin panel"
      }), /* @__PURE__ */ jsx(Separator, {
        className: "hidden"
      })]
    })
  });
});
const route1 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  FileText,
  default: shell
}, Symbol.toStringTag, { value: "Module" }));
const Card = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("card-vc text-card-foreground", className),
    ...props
  }
));
Card.displayName = "Card";
const CardHeader = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("flex flex-col space-y-1.5 p-6", className),
    ...props
  }
));
CardHeader.displayName = "CardHeader";
const CardTitle = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("font-semibold leading-none tracking-tight", className),
    ...props
  }
));
CardTitle.displayName = "CardTitle";
const CardDescription = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("text-sm text-muted-foreground", className),
    ...props
  }
));
CardDescription.displayName = "CardDescription";
const CardContent = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx("div", { ref, className: cn("p-6 pt-0", className), ...props }));
CardContent.displayName = "CardContent";
const CardFooter = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("flex items-center p-6 pt-0", className),
    ...props
  }
));
CardFooter.displayName = "CardFooter";
function Skeleton({ className, ...props }) {
  return /* @__PURE__ */ jsx(
    "div",
    {
      className: cn("animate-pulse rounded-md bg-muted", className),
      ...props
    }
  );
}
function CountUp({
  value,
  suffix = ""
}) {
  const mv = useMotionValue(0);
  const display = useTransform(mv, (latest) => Math.round(latest).toString());
  const [text, setText] = useState("0");
  useEffect(() => {
    const controls = animate(mv, value, {
      duration: 0.8,
      ease: "easeOut"
    });
    const unsub = display.on("change", (v) => setText(v));
    return () => {
      controls.stop();
      unsub();
    };
  }, [value, mv, display]);
  return /* @__PURE__ */ jsxs("span", {
    className: "font-mono tabular-nums",
    children: [text, suffix]
  });
}
function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  accent
}) {
  return /* @__PURE__ */ jsxs(Card, {
    className: "relative overflow-hidden",
    children: [/* @__PURE__ */ jsx("div", {
      className: "pointer-events-none absolute inset-0 opacity-30",
      style: {
        background: `radial-gradient(60% 80% at 0% 0%, ${accent} 0%, transparent 60%)`
      }
    }), /* @__PURE__ */ jsxs(CardHeader, {
      className: "flex flex-row items-center justify-between space-y-0 pb-2",
      children: [/* @__PURE__ */ jsx(CardTitle, {
        className: "text-xs font-medium uppercase tracking-wide text-muted-foreground",
        children: label
      }), /* @__PURE__ */ jsx(Icon, {
        className: "h-4 w-4 text-muted-foreground"
      })]
    }), /* @__PURE__ */ jsxs(CardContent, {
      children: [/* @__PURE__ */ jsx("div", {
        className: "text-3xl font-semibold",
        children: /* @__PURE__ */ jsx(CountUp, {
          value
        })
      }), /* @__PURE__ */ jsx("p", {
        className: "mt-1 text-xs text-muted-foreground",
        children: hint
      })]
    })]
  });
}
function SkeletonStat() {
  return /* @__PURE__ */ jsxs(Card, {
    children: [/* @__PURE__ */ jsxs(CardHeader, {
      className: "flex flex-row items-center justify-between space-y-0 pb-2",
      children: [/* @__PURE__ */ jsx(Skeleton, {
        className: "h-3 w-20"
      }), /* @__PURE__ */ jsx(Skeleton, {
        className: "h-4 w-4 rounded"
      })]
    }), /* @__PURE__ */ jsxs(CardContent, {
      children: [/* @__PURE__ */ jsx(Skeleton, {
        className: "h-8 w-16"
      }), /* @__PURE__ */ jsx(Skeleton, {
        className: "mt-2 h-3 w-32"
      })]
    })]
  });
}
const _index = UNSAFE_withComponentProps(function Dashboard() {
  var _a;
  const {
    data,
    isLoading
  } = useQuery({
    queryKey: ["status"],
    queryFn: () => api("/api/status"),
    refetchInterval: 5e3
  });
  const installed = data ? data.compose_services.filter((s) => s.state === "running").length : 0;
  const registry = 16;
  const selected = (data == null ? void 0 : data.state.selected.length) ?? 0;
  const services2 = (data == null ? void 0 : data.compose_services.length) ?? 0;
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-8",
    children: [/* @__PURE__ */ jsxs(motion.div, {
      initial: {
        opacity: 0,
        y: 8
      },
      animate: {
        opacity: 1,
        y: 0
      },
      transition: {
        duration: 0.4,
        ease: "easeOut"
      },
      className: "space-y-2",
      children: [/* @__PURE__ */ jsxs(Badge, {
        variant: "outline",
        className: "gap-2",
        children: [/* @__PURE__ */ jsx(Sparkles, {
          className: "h-3 w-3"
        }), "Panel de control"]
      }), /* @__PURE__ */ jsxs("h1", {
        className: "text-3xl font-semibold tracking-tight",
        children: ["Hola", ((_a = data == null ? void 0 : data.env_mem0) == null ? void 0 : _a.MEM0_USER_ID) ? `, ${data.env_mem0.MEM0_USER_ID}` : ""]
      }), /* @__PURE__ */ jsx("p", {
        className: "text-muted-foreground",
        children: "Tu stack MCP en un vistazo. Las cifras se actualizan cada 5 segundos."
      })]
    }), /* @__PURE__ */ jsx("div", {
      className: "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4",
      children: isLoading || !data ? /* @__PURE__ */ jsxs(Fragment, {
        children: [/* @__PURE__ */ jsx(SkeletonStat, {}), /* @__PURE__ */ jsx(SkeletonStat, {}), /* @__PURE__ */ jsx(SkeletonStat, {}), /* @__PURE__ */ jsx(SkeletonStat, {})]
      }) : /* @__PURE__ */ jsxs(Fragment, {
        children: [/* @__PURE__ */ jsx(StatCard, {
          label: "Tools seleccionadas",
          value: selected,
          icon: Box,
          hint: "Componentes activos del state.json",
          accent: "#a855f7"
        }), /* @__PURE__ */ jsx(StatCard, {
          label: "Servicios corriendo",
          value: installed,
          icon: Container,
          hint: `${services2} servicios docker totales`,
          accent: "#38bdf8"
        }), /* @__PURE__ */ jsx(StatCard, {
          label: "En el registro",
          value: registry,
          icon: Activity,
          hint: "Componentes disponibles para instalar",
          accent: "#ec4899"
        }), /* @__PURE__ */ jsx(StatCard, {
          label: "Última actualización",
          value: 0,
          icon: Sparkles,
          hint: data.state.updated_at ? new Date(data.state.updated_at).toLocaleString("es-ES") : "—",
          accent: "#fbbf24"
        })]
      })
    }), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsx(CardTitle, {
          children: "Servicios docker"
        }), /* @__PURE__ */ jsx(CardDescription, {
          children: "Estado en vivo de los contenedores definidos en dockers/compose.yaml."
        })]
      }), /* @__PURE__ */ jsx(CardContent, {
        children: isLoading || !data ? /* @__PURE__ */ jsx("div", {
          className: "grid gap-2 sm:grid-cols-2",
          children: [0, 1, 2, 3].map((i) => /* @__PURE__ */ jsx(Skeleton, {
            className: "h-10 w-full"
          }, i))
        }) : data.compose_services.length === 0 ? /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Aún no hay servicios docker."
        }) : /* @__PURE__ */ jsx("div", {
          className: "grid gap-2 sm:grid-cols-2",
          children: data.compose_services.map((svc) => /* @__PURE__ */ jsxs("div", {
            className: "flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2",
            children: [/* @__PURE__ */ jsx("span", {
              className: "font-mono text-sm",
              children: svc.name
            }), /* @__PURE__ */ jsx(Badge, {
              variant: svc.state === "running" ? "success" : "secondary",
              children: svc.state
            })]
          }, svc.name))
        })
      })]
    })]
  });
});
const route2 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: _index
}, Symbol.toStringTag, { value: "Module" }));
function useJobStream(jobId) {
  const [state, setState] = useState({
    lines: [],
    done: false,
    ok: false,
    error: null,
    open: false
  });
  useRef(null);
  useEffect(() => {
    if (!jobId) return;
    let cancelled = false;
    let reader = null;
    let buf = "";
    (async () => {
      try {
        const res = await apiStream(`/api/jobs/${jobId}/events`);
        if (!res.ok || !res.body) {
          setState((s) => ({
            ...s,
            done: true,
            ok: false,
            error: `SSE handshake failed: ${res.status}`,
            open: false
          }));
          return;
        }
        setState((s) => ({ ...s, open: true }));
        reader = res.body.getReader();
        const dec = new TextDecoder();
        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += dec.decode(value, { stream: true });
          let idx;
          while ((idx = buf.indexOf("\n\n")) !== -1) {
            const frame = buf.slice(0, idx);
            buf = buf.slice(idx + 2);
            handleFrame(frame);
          }
        }
        setState((s) => ({ ...s, open: false }));
      } catch (err) {
        setState((s) => ({
          ...s,
          done: true,
          ok: false,
          error: err instanceof Error ? err.message : String(err),
          open: false
        }));
      }
    })();
    function handleFrame(frame) {
      let event = "message";
      const dataLines = [];
      for (const line of frame.split("\n")) {
        if (line.startsWith("event:")) {
          event = line.slice(6).trim();
        } else if (line.startsWith("data:")) {
          dataLines.push(line.slice(5).trim());
        }
      }
      const data = dataLines.join("\n");
      if (event === "done") {
        let ok = true;
        let error;
        try {
          const parsed = JSON.parse(data);
          ok = !!parsed.ok;
          error = parsed.error;
        } catch {
        }
        setState((s) => ({
          ...s,
          done: true,
          ok,
          error: error ?? (ok ? null : s.error),
          open: false
        }));
        return;
      }
      if (!data) return;
      const sp = data.indexOf(" ");
      if (sp === -1) {
        setState((s) => ({
          ...s,
          lines: [...s.lines, { stream: "system", text: data }]
        }));
      } else {
        const stream = data.slice(0, sp);
        const text = data.slice(sp + 1);
        if (stream === "stdout" || stream === "stderr" || stream === "system") {
          setState((s) => ({
            ...s,
            lines: [...s.lines, { stream, text }]
          }));
        }
      }
    }
    return () => {
      cancelled = true;
      if (reader) {
        reader.cancel().catch(() => void 0);
      }
    };
  }, [jobId]);
  const reset = useCallback(() => {
    setState({ lines: [], done: false, ok: false, error: null, open: false });
  }, []);
  return { ...state, reset };
}
function useEventSource(url, onMessage) {
  useEffect(() => {
    if (!url) return;
    let cancelled = false;
    let reader = null;
    let buf = "";
    (async () => {
      try {
        const res = await apiStream(url);
        if (!res.ok || !res.body) return;
        reader = res.body.getReader();
        const dec = new TextDecoder();
        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += dec.decode(value, { stream: true });
          let idx;
          while ((idx = buf.indexOf("\n\n")) !== -1) {
            const frame = buf.slice(0, idx);
            buf = buf.slice(idx + 2);
            for (const line of frame.split("\n")) {
              if (line.startsWith("data:")) {
                onMessage(line.slice(5).trim());
              }
            }
          }
        }
      } catch {
      }
    })();
    return () => {
      cancelled = true;
      if (reader) reader.cancel().catch(() => void 0);
    };
  }, [url, onMessage]);
}
const Switch = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  SwitchPrimitive.Root,
  {
    ref,
    className: cn(
      "peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input",
      className
    ),
    ...props,
    children: /* @__PURE__ */ jsx(
      SwitchPrimitive.Thumb,
      {
        className: cn(
          "pointer-events-none block h-4 w-4 rounded-full bg-background shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0"
        )
      }
    )
  }
));
Switch.displayName = SwitchPrimitive.Root.displayName;
const Dialog = DialogPrimitive.Root;
const DialogPortal = DialogPrimitive.Portal;
const DialogOverlay = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  DialogPrimitive.Overlay,
  {
    ref,
    className: cn(
      "fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      className
    ),
    ...props
  }
));
DialogOverlay.displayName = DialogPrimitive.Overlay.displayName;
const DialogContent = React.forwardRef(({ className, children, ...props }, ref) => /* @__PURE__ */ jsxs(DialogPortal, { children: [
  /* @__PURE__ */ jsx(DialogOverlay, {}),
  /* @__PURE__ */ jsxs(
    DialogPrimitive.Content,
    {
      ref,
      className: cn(
        "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border border-border bg-background p-6 shadow-lg duration-200 sm:rounded-lg",
        className
      ),
      ...props,
      children: [
        children,
        /* @__PURE__ */ jsxs(DialogPrimitive.Close, { className: "absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none", children: [
          /* @__PURE__ */ jsx(X, { className: "h-4 w-4" }),
          /* @__PURE__ */ jsx("span", { className: "sr-only", children: "Cerrar" })
        ] })
      ]
    }
  )
] }));
DialogContent.displayName = DialogPrimitive.Content.displayName;
const DialogHeader = ({
  className,
  ...props
}) => /* @__PURE__ */ jsx(
  "div",
  {
    className: cn("flex flex-col space-y-1.5 text-left", className),
    ...props
  }
);
DialogHeader.displayName = "DialogHeader";
const DialogFooter = ({
  className,
  ...props
}) => /* @__PURE__ */ jsx(
  "div",
  {
    className: cn(
      "flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2",
      className
    ),
    ...props
  }
);
DialogFooter.displayName = "DialogFooter";
const DialogTitle = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  DialogPrimitive.Title,
  {
    ref,
    className: cn("text-lg font-semibold leading-none tracking-tight", className),
    ...props
  }
));
DialogTitle.displayName = DialogPrimitive.Title.displayName;
const DialogDescription = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  DialogPrimitive.Description,
  {
    ref,
    className: cn("text-sm text-muted-foreground", className),
    ...props
  }
));
DialogDescription.displayName = DialogPrimitive.Description.displayName;
const alertVariants = cva(
  "relative w-full rounded-lg border p-4 [&>svg~*]:pl-7 [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground",
  {
    variants: {
      variant: {
        default: "bg-background text-foreground",
        destructive: "border-destructive/50 text-destructive [&>svg]:text-destructive bg-destructive/10",
        warning: "border-amber-500/50 text-amber-300 [&>svg]:text-amber-300 bg-amber-500/10"
      }
    },
    defaultVariants: { variant: "default" }
  }
);
const Alert = React.forwardRef(({ className, variant, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    role: "alert",
    className: cn(alertVariants({ variant }), className),
    ...props
  }
));
Alert.displayName = "Alert";
const AlertTitle = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "h5",
  {
    ref,
    className: cn("mb-1 font-medium leading-none tracking-tight", className),
    ...props
  }
));
AlertTitle.displayName = "AlertTitle";
const AlertDescription = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  "div",
  {
    ref,
    className: cn("text-sm [&_p]:leading-relaxed", className),
    ...props
  }
));
AlertDescription.displayName = "AlertDescription";
function statusVariant(view) {
  if (view.installed) return "success";
  if (view.deploy === "Sudo") return "warning";
  return "secondary";
}
function statusLabel(view) {
  if (view.installed && view.version) return `v${view.version}`;
  if (view.installed) return "instalado";
  if (view.deploy === "Sudo") return "requiere sudo";
  return "no instalado";
}
function runAction(action, key, body) {
  const path = `/api/tools/${encodeURIComponent(key)}/${action}`;
  return api(path, {
    method: "POST",
    body
  });
}
function RunDialog({
  toolKey,
  toolLabel,
  action,
  jobId,
  open,
  onOpenChange
}) {
  const job = useJobStream(jobId);
  return /* @__PURE__ */ jsx(Dialog, {
    open,
    onOpenChange,
    children: /* @__PURE__ */ jsxs(DialogContent, {
      className: "max-w-2xl",
      children: [/* @__PURE__ */ jsxs(DialogHeader, {
        children: [/* @__PURE__ */ jsxs(DialogTitle, {
          children: [action, " · ", toolKey]
        }), /* @__PURE__ */ jsxs(DialogDescription, {
          children: [toolLabel, " · job ", jobId ?? "—"]
        })]
      }), /* @__PURE__ */ jsxs("div", {
        className: "max-h-80 overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs",
        children: [job.lines.length === 0 && job.open && /* @__PURE__ */ jsx("p", {
          className: "text-muted-foreground",
          children: "Iniciando…"
        }), job.lines.map((l, i) => /* @__PURE__ */ jsx("div", {
          className: l.stream === "stderr" ? "text-amber-300" : "text-foreground/90",
          children: l.text
        }, i)), job.done && /* @__PURE__ */ jsx("div", {
          className: job.ok ? "mt-2 text-emerald-300" : "mt-2 text-red-300",
          children: job.ok ? "✓ completado" : `✗ ${job.error ?? "falló"}`
        })]
      }), /* @__PURE__ */ jsx(DialogFooter, {
        children: /* @__PURE__ */ jsx(Button, {
          variant: "outline",
          onClick: () => onOpenChange(false),
          children: "Cerrar"
        })
      })]
    })
  });
}
function ToolRow({
  view
}) {
  const qc = useQueryClient();
  const [pending, setPending] = useState(null);
  const [jobId, setJobId] = useState(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [error, setError] = useState(null);
  const mutate = useMutation({
    mutationFn: (vars) => runAction(vars.action, view.key, vars.body),
    onSuccess: (res, vars) => {
      setPending(null);
      setJobId(res.job_id);
      setDialogOpen(true);
      toast.success(`${vars.action} ${view.key} encolado`, {
        description: `job ${res.job_id}`
      });
    },
    onError: (err, vars) => {
      setPending(null);
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
      toast.error(`No se pudo ${vars.action} ${view.key}`, {
        description: msg
      });
    },
    onSettled: () => {
      qc.invalidateQueries({
        queryKey: ["tools"]
      });
      qc.invalidateQueries({
        queryKey: ["status"]
      });
    }
  });
  function start(action, body) {
    setError(null);
    setPending(action);
    mutate.mutate({
      action,
      body
    });
  }
  return /* @__PURE__ */ jsxs(motion.div, {
    layout: true,
    children: [/* @__PURE__ */ jsx(Card, {
      className: "border-border/60",
      children: /* @__PURE__ */ jsxs(CardContent, {
        className: "grid gap-4 py-5 md:grid-cols-[1fr_auto]",
        children: [/* @__PURE__ */ jsxs("div", {
          className: "space-y-2",
          children: [/* @__PURE__ */ jsxs("div", {
            className: "flex items-center gap-2",
            children: [/* @__PURE__ */ jsx("span", {
              className: "font-mono text-sm font-semibold",
              children: view.key
            }), /* @__PURE__ */ jsx(Badge, {
              variant: statusVariant(view),
              children: statusLabel(view)
            }), /* @__PURE__ */ jsx(Badge, {
              variant: "outline",
              children: view.deploy
            })]
          }), /* @__PURE__ */ jsx("p", {
            className: "text-sm font-medium",
            children: view.label
          }), /* @__PURE__ */ jsx("p", {
            className: "text-xs text-muted-foreground",
            children: view.summary
          }), error && /* @__PURE__ */ jsxs(Alert, {
            variant: "destructive",
            className: "py-2",
            children: [/* @__PURE__ */ jsx(AlertCircle, {
              className: "h-4 w-4"
            }), /* @__PURE__ */ jsx(AlertTitle, {
              className: "text-xs",
              children: "Error"
            }), /* @__PURE__ */ jsx(AlertDescription, {
              className: "text-xs",
              children: error
            })]
          })]
        }), /* @__PURE__ */ jsxs("div", {
          className: "flex flex-wrap items-center justify-end gap-2",
          children: [/* @__PURE__ */ jsxs("div", {
            className: "flex items-center gap-2",
            children: [/* @__PURE__ */ jsx("span", {
              className: "text-xs text-muted-foreground",
              children: "selected"
            }), /* @__PURE__ */ jsx(Switch, {
              checked: view.selected,
              disabled: true
            })]
          }), /* @__PURE__ */ jsxs(Button, {
            size: "sm",
            variant: "outline",
            disabled: pending !== null,
            onClick: () => start("install"),
            children: [pending === "install" ? /* @__PURE__ */ jsx(Loader2, {
              className: "h-3 w-3 animate-spin"
            }) : /* @__PURE__ */ jsx(Download, {
              className: "h-3 w-3"
            }), "install"]
          }), /* @__PURE__ */ jsxs(Button, {
            size: "sm",
            variant: "outline",
            disabled: pending !== null || !view.installed,
            onClick: () => start("upgrade"),
            children: [pending === "upgrade" ? /* @__PURE__ */ jsx(Loader2, {
              className: "h-3 w-3 animate-spin"
            }) : /* @__PURE__ */ jsx(RefreshCcw, {
              className: "h-3 w-3"
            }), "upgrade"]
          }), /* @__PURE__ */ jsxs(Button, {
            size: "sm",
            variant: "outline",
            disabled: pending !== null || !view.installed,
            onClick: () => start("uninstall", {
              force: false
            }),
            children: [pending === "uninstall" ? /* @__PURE__ */ jsx(Loader2, {
              className: "h-3 w-3 animate-spin"
            }) : /* @__PURE__ */ jsx(Trash2, {
              className: "h-3 w-3"
            }), "uninstall"]
          })]
        })]
      })
    }), /* @__PURE__ */ jsx(AnimatePresence, {
      children: dialogOpen && /* @__PURE__ */ jsx(RunDialog, {
        toolKey: view.key,
        toolLabel: view.label,
        action: pending ?? "install",
        jobId,
        open: dialogOpen,
        onOpenChange: setDialogOpen
      })
    })]
  });
}
const tools = UNSAFE_withComponentProps(function ToolsRoute() {
  const {
    data,
    isLoading,
    error
  } = useQuery({
    queryKey: ["tools"],
    queryFn: () => api("/api/tools"),
    refetchInterval: 5e3
  });
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-center justify-between",
      children: [/* @__PURE__ */ jsxs("div", {
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-2xl font-semibold tracking-tight",
          children: "Tools"
        }), /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Componentes del stack MCP. Instala, actualiza o desinstala uno a uno."
        })]
      }), /* @__PURE__ */ jsxs(Badge, {
        variant: "outline",
        children: [(data == null ? void 0 : data.length) ?? 0, " totales"]
      })]
    }), isLoading && /* @__PURE__ */ jsx("div", {
      className: "grid gap-3",
      children: [0, 1, 2].map((i) => /* @__PURE__ */ jsx("div", {
        className: "h-20 rounded-lg border border-border/40 bg-card/40"
      }, i))
    }), error && /* @__PURE__ */ jsxs(Alert, {
      variant: "destructive",
      children: [/* @__PURE__ */ jsx(AlertCircle, {
        className: "h-4 w-4"
      }), /* @__PURE__ */ jsx(AlertTitle, {
        children: "Error"
      }), /* @__PURE__ */ jsx(AlertDescription, {
        children: error.message ?? "no se pudo cargar /api/tools"
      })]
    }), data && /* @__PURE__ */ jsxs("div", {
      className: "grid gap-3",
      children: [data.map((v) => /* @__PURE__ */ jsx(ToolRow, {
        view: v
      }, v.key)), data.length === 0 && /* @__PURE__ */ jsx(Card, {
        children: /* @__PURE__ */ jsx(CardHeader, {
          children: /* @__PURE__ */ jsx(CardTitle, {
            className: "text-base",
            children: "Sin tools en el registro"
          })
        })
      })]
    }), /* @__PURE__ */ jsxs("div", {
      className: "hidden",
      children: [/* @__PURE__ */ jsx(CheckCircle2, {}), /* @__PURE__ */ jsx(XCircle, {})]
    })]
  });
});
const route3 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: tools
}, Symbol.toStringTag, { value: "Module" }));
const configure = UNSAFE_withComponentProps(function ConfigureRoute() {
  const qc = useQueryClient();
  const {
    data: tools2,
    isLoading
  } = useQuery({
    queryKey: ["tools"],
    queryFn: () => api("/api/tools")
  });
  const [selected, setSelected] = useState(null);
  const [error, setError] = useState(null);
  const chosen = useMemo(() => {
    if (selected) return selected;
    if (!tools2) return /* @__PURE__ */ new Set();
    return new Set(tools2.filter((t) => t.selected).map((t) => t.key));
  }, [tools2, selected]);
  function toggle(key) {
    setSelected(new Set(chosen));
    const next = new Set(chosen);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setSelected(next);
  }
  const mutate = useMutation({
    mutationFn: () => api("/api/configure", {
      method: "POST",
      body: {
        selected: Array.from(chosen)
      }
    }),
    onSuccess: (res) => {
      toast.success("Configuración encolada", {
        description: `job ${res.job_id}`
      });
      qc.invalidateQueries({
        queryKey: ["tools"]
      });
      qc.invalidateQueries({
        queryKey: ["status"]
      });
    },
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
      toast.error("No se pudo aplicar la selección", {
        description: msg
      });
    }
  });
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-end justify-between",
      children: [/* @__PURE__ */ jsxs("div", {
        className: "space-y-1",
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-2xl font-semibold tracking-tight",
          children: "Configurar selección"
        }), /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Marca o desmarca los componentes. El orquestador respeta las dependencias declaradas."
        })]
      }), /* @__PURE__ */ jsxs(Button, {
        disabled: mutate.isPending || isLoading || !tools2,
        onClick: () => mutate.mutate(),
        children: [mutate.isPending ? /* @__PURE__ */ jsx(Loader2, {
          className: "h-4 w-4 animate-spin"
        }) : /* @__PURE__ */ jsx(Save, {
          className: "h-4 w-4"
        }), "Aplicar cambios"]
      })]
    }), error && /* @__PURE__ */ jsxs(Alert, {
      variant: "destructive",
      children: [/* @__PURE__ */ jsx(AlertTitle, {
        children: "Error"
      }), /* @__PURE__ */ jsx(AlertDescription, {
        children: error
      })]
    }), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsxs(CardTitle, {
          className: "flex items-center gap-2 text-base",
          children: [/* @__PURE__ */ jsx(Layers, {
            className: "h-4 w-4"
          }), "Selección actual"]
        }), /* @__PURE__ */ jsxs(CardDescription, {
          children: [chosen.size, " tool", chosen.size === 1 ? "" : "s", " marcados"]
        })]
      }), /* @__PURE__ */ jsx(CardContent, {
        className: "grid gap-2",
        children: isLoading || !tools2 ? /* @__PURE__ */ jsx("div", {
          className: "text-sm text-muted-foreground",
          children: "Cargando…"
        }) : tools2.map((t) => /* @__PURE__ */ jsxs(motion.div, {
          layout: true,
          className: "flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2",
          children: [/* @__PURE__ */ jsxs("div", {
            className: "flex items-center gap-3",
            children: [/* @__PURE__ */ jsx(Switch, {
              checked: chosen.has(t.key),
              onCheckedChange: () => toggle(t.key)
            }), /* @__PURE__ */ jsxs("div", {
              children: [/* @__PURE__ */ jsxs("div", {
                className: "flex items-center gap-2 text-sm",
                children: [/* @__PURE__ */ jsx("span", {
                  className: "font-mono",
                  children: t.key
                }), /* @__PURE__ */ jsx(Badge, {
                  variant: "outline",
                  children: t.deploy
                })]
              }), /* @__PURE__ */ jsx("p", {
                className: "text-xs text-muted-foreground",
                children: t.summary
              })]
            })]
          }), t.deps.length > 0 && /* @__PURE__ */ jsxs("span", {
            className: "text-xs text-muted-foreground",
            children: ["deps: ", t.deps.join(", ")]
          })]
        }, t.key))
      })]
    })]
  });
});
const route4 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: configure
}, Symbol.toStringTag, { value: "Module" }));
const Input = React.forwardRef(
  ({ className, type, ...props }, ref) => /* @__PURE__ */ jsx(
    "input",
    {
      type,
      ref,
      className: cn(
        "flex h-9 w-full rounded-md border border-border bg-background px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50",
        className
      ),
      ...props
    }
  )
);
Input.displayName = "Input";
function PullDialog({
  tag,
  jobId,
  open,
  onOpenChange
}) {
  const job = useJobStream(jobId);
  return /* @__PURE__ */ jsx(Dialog, {
    open,
    onOpenChange,
    children: /* @__PURE__ */ jsxs(DialogContent, {
      className: "max-w-2xl",
      children: [/* @__PURE__ */ jsxs(DialogHeader, {
        children: [/* @__PURE__ */ jsxs(DialogTitle, {
          children: ["pull ", tag ?? ""]
        }), /* @__PURE__ */ jsxs(DialogDescription, {
          children: ["job ", jobId ?? "—"]
        })]
      }), /* @__PURE__ */ jsxs("div", {
        className: "max-h-80 overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs",
        children: [job.lines.length === 0 && /* @__PURE__ */ jsx("span", {
          className: "text-muted-foreground",
          children: "Iniciando…"
        }), job.lines.map((l, i) => /* @__PURE__ */ jsx("div", {
          className: l.stream === "stderr" ? "text-amber-300" : "text-foreground/90",
          children: l.text
        }, i)), job.done && /* @__PURE__ */ jsx("div", {
          className: job.ok ? "mt-2 text-emerald-300" : "mt-2 text-red-300",
          children: job.ok ? "✓ listo" : `✗ ${job.error ?? "falló"}`
        })]
      })]
    })
  });
}
const models = UNSAFE_withComponentProps(function ModelsRoute() {
  const qc = useQueryClient();
  const {
    data,
    isLoading,
    error
  } = useQuery({
    queryKey: ["models"],
    queryFn: () => api("/api/models"),
    refetchInterval: 5e3
  });
  const [tag, setTag] = useState("");
  const [activeJob, setActiveJob] = useState(null);
  const [errMsg, setErrMsg] = useState(null);
  const pullMut = useMutation({
    mutationFn: (t) => api("/api/models/pull", {
      method: "POST",
      body: {
        tag: t
      }
    }),
    onSuccess: (res, t) => {
      setActiveJob({
        tag: t,
        id: res.job_id
      });
      toast.success(`pull ${t} encolado`, {
        description: `job ${res.job_id}`
      });
      qc.invalidateQueries({
        queryKey: ["models"]
      });
    },
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error("pull falló", {
        description: msg
      });
    }
  });
  const rmMut = useMutation({
    mutationFn: (t) => api("/api/models/rm", {
      method: "POST",
      body: {
        tag: t
      }
    }),
    onSuccess: (res, t) => {
      toast.success(`rm ${t} encolado`, {
        description: `job ${res.job_id}`
      });
      qc.invalidateQueries({
        queryKey: ["models"]
      });
    },
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error("rm falló", {
        description: msg
      });
    }
  });
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-end justify-between",
      children: [/* @__PURE__ */ jsxs("div", {
        className: "space-y-1",
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-2xl font-semibold tracking-tight",
          children: "Modelos Ollama"
        }), /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Descarga o elimina modelos del contenedor mcp-tools-ollama."
        })]
      }), /* @__PURE__ */ jsxs("form", {
        className: "flex items-center gap-2",
        onSubmit: (e) => {
          e.preventDefault();
          const t = tag.trim();
          if (!t) return;
          pullMut.mutate(t);
          setTag("");
        },
        children: [/* @__PURE__ */ jsx(Input, {
          value: tag,
          onChange: (e) => setTag(e.target.value),
          placeholder: "qwen2.5:7b",
          className: "w-44"
        }), /* @__PURE__ */ jsxs(Button, {
          type: "submit",
          disabled: pullMut.isPending || tag.trim() === "",
          children: [pullMut.isPending ? /* @__PURE__ */ jsx(Loader2, {
            className: "h-4 w-4 animate-spin"
          }) : /* @__PURE__ */ jsx(Download, {
            className: "h-4 w-4"
          }), "Pull"]
        })]
      })]
    }), errMsg && /* @__PURE__ */ jsxs(Alert, {
      variant: "destructive",
      children: [/* @__PURE__ */ jsx(AlertTitle, {
        children: "Error"
      }), /* @__PURE__ */ jsx(AlertDescription, {
        children: errMsg
      })]
    }), error && /* @__PURE__ */ jsxs(Alert, {
      variant: "destructive",
      children: [/* @__PURE__ */ jsx(AlertTitle, {
        children: "Error"
      }), /* @__PURE__ */ jsx(AlertDescription, {
        children: error.message ?? "no se pudo cargar /api/models"
      })]
    }), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsxs(CardTitle, {
          className: "flex items-center gap-2 text-base",
          children: [/* @__PURE__ */ jsx(Database, {
            className: "h-4 w-4"
          }), "Instalados"]
        }), /* @__PURE__ */ jsx(CardDescription, {
          children: isLoading ? "Cargando…" : `${(data == null ? void 0 : data.length) ?? 0} modelo${(data == null ? void 0 : data.length) === 1 ? "" : "s"}`
        })]
      }), /* @__PURE__ */ jsx(CardContent, {
        className: "grid gap-2",
        children: (data == null ? void 0 : data.length) ? data.map((m) => /* @__PURE__ */ jsxs(motion.div, {
          layout: true,
          className: "flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2",
          children: [/* @__PURE__ */ jsxs("div", {
            className: "flex items-center gap-3",
            children: [/* @__PURE__ */ jsx("span", {
              className: "font-mono text-sm",
              children: m.tag
            }), /* @__PURE__ */ jsx(Badge, {
              variant: "outline",
              children: m.size
            }), /* @__PURE__ */ jsx("span", {
              className: "text-xs text-muted-foreground",
              children: m.modified
            })]
          }), /* @__PURE__ */ jsxs(Button, {
            size: "sm",
            variant: "outline",
            disabled: rmMut.isPending,
            onClick: () => rmMut.mutate(m.tag),
            children: [rmMut.isPending && rmMut.variables === m.tag ? /* @__PURE__ */ jsx(Loader2, {
              className: "h-3 w-3 animate-spin"
            }) : /* @__PURE__ */ jsx(Trash2, {
              className: "h-3 w-3"
            }), "rm"]
          })]
        }, m.tag)) : /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Sin modelos instalados."
        })
      })]
    }), activeJob && /* @__PURE__ */ jsx(PullDialog, {
      tag: activeJob.tag,
      jobId: activeJob.id,
      open: true,
      onOpenChange: (next) => {
        if (!next) setActiveJob(null);
      }
    })]
  });
});
const route5 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: models
}, Symbol.toStringTag, { value: "Module" }));
function LogsDialog({
  service,
  open,
  onOpenChange
}) {
  const [lines, setLines] = useState([]);
  useEventSource(service ? `/api/logs/${encodeURIComponent(service)}?tail=80&follow=1` : null, (line) => {
    setLines((prev) => {
      const next = [...prev, line];
      return next.length > 500 ? next.slice(next.length - 500) : next;
    });
  });
  return /* @__PURE__ */ jsx(Dialog, {
    open,
    onOpenChange,
    children: /* @__PURE__ */ jsxs(DialogContent, {
      className: "max-w-3xl",
      children: [/* @__PURE__ */ jsxs(DialogHeader, {
        children: [/* @__PURE__ */ jsxs(DialogTitle, {
          children: ["logs · ", service]
        }), /* @__PURE__ */ jsx(DialogDescription, {
          children: "Stream en vivo de docker logs"
        })]
      }), /* @__PURE__ */ jsx("pre", {
        className: "max-h-[60vh] overflow-y-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs whitespace-pre-wrap",
        children: lines.join("\n") || "Esperando salida…"
      })]
    })
  });
}
const services = UNSAFE_withComponentProps(function ServicesRoute() {
  const qc = useQueryClient();
  const {
    data,
    isLoading,
    error
  } = useQuery({
    queryKey: ["services"],
    queryFn: () => api("/api/services"),
    refetchInterval: 5e3
  });
  const [logsSvc, setLogsSvc] = useState(null);
  const [errMsg, setErrMsg] = useState(null);
  const ctrlMut = useMutation({
    mutationFn: ({
      name,
      verb
    }) => api(`/api/services/${encodeURIComponent(name)}/${verb}`, {
      method: "POST"
    }),
    onSuccess: (res, vars) => {
      toast.success(`${vars.verb} ${vars.name}`, {
        description: `job ${res.job_id}`
      });
      qc.invalidateQueries({
        queryKey: ["services"]
      });
      qc.invalidateQueries({
        queryKey: ["status"]
      });
    },
    onError: (err, vars) => {
      const msg = err instanceof Error ? err.message : String(err);
      setErrMsg(msg);
      toast.error(`${vars.verb} ${vars.name} falló`, {
        description: msg
      });
    }
  });
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsxs("div", {
      className: "flex items-end justify-between",
      children: [/* @__PURE__ */ jsxs("div", {
        className: "space-y-1",
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-2xl font-semibold tracking-tight",
          children: "Servicios docker"
        }), /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Arranca, para o reinicia cada servicio del compose."
        })]
      }), /* @__PURE__ */ jsxs(Badge, {
        variant: "outline",
        children: [/* @__PURE__ */ jsx(Cog, {
          className: "mr-1 h-3 w-3"
        }), " ", (data == null ? void 0 : data.length) ?? 0, " servicios"]
      })]
    }), errMsg && /* @__PURE__ */ jsx(Card, {
      className: "border-destructive/40",
      children: /* @__PURE__ */ jsx(CardContent, {
        className: "py-3 text-sm text-destructive",
        children: errMsg
      })
    }), error && /* @__PURE__ */ jsx(Card, {
      children: /* @__PURE__ */ jsx(CardContent, {
        className: "py-3 text-sm text-destructive",
        children: error.message
      })
    }), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsx(CardTitle, {
          className: "text-base",
          children: "Servicios definidos"
        }), /* @__PURE__ */ jsx(CardDescription, {
          children: isLoading ? "Cargando…" : "Acciones inmediatas vía docker compose."
        })]
      }), /* @__PURE__ */ jsxs(CardContent, {
        className: "grid gap-2",
        children: [data == null ? void 0 : data.map((svc) => {
          var _a;
          return /* @__PURE__ */ jsxs(motion.div, {
            layout: true,
            className: "flex items-center justify-between rounded-md border border-border/60 bg-background/40 px-3 py-2",
            children: [/* @__PURE__ */ jsxs("div", {
              className: "flex items-center gap-3",
              children: [/* @__PURE__ */ jsx("span", {
                className: "font-mono text-sm",
                children: svc.name
              }), /* @__PURE__ */ jsx(Badge, {
                variant: svc.state === "running" ? "success" : "secondary",
                children: svc.state
              })]
            }), /* @__PURE__ */ jsxs("div", {
              className: "flex items-center gap-1",
              children: [/* @__PURE__ */ jsxs(Button, {
                size: "sm",
                variant: "outline",
                disabled: ctrlMut.isPending,
                onClick: () => ctrlMut.mutate({
                  name: svc.name,
                  verb: "up"
                }),
                children: [/* @__PURE__ */ jsx(Play, {
                  className: "h-3 w-3"
                }), " up"]
              }), /* @__PURE__ */ jsxs(Button, {
                size: "sm",
                variant: "outline",
                disabled: ctrlMut.isPending,
                onClick: () => ctrlMut.mutate({
                  name: svc.name,
                  verb: "stop"
                }),
                children: [/* @__PURE__ */ jsx(Square, {
                  className: "h-3 w-3"
                }), " stop"]
              }), /* @__PURE__ */ jsxs(Button, {
                size: "sm",
                variant: "outline",
                disabled: ctrlMut.isPending,
                onClick: () => ctrlMut.mutate({
                  name: svc.name,
                  verb: "restart"
                }),
                children: [ctrlMut.isPending && ((_a = ctrlMut.variables) == null ? void 0 : _a.name) === svc.name ? /* @__PURE__ */ jsx(Loader2, {
                  className: "h-3 w-3 animate-spin"
                }) : /* @__PURE__ */ jsx(RotateCcw, {
                  className: "h-3 w-3"
                }), "restart"]
              }), /* @__PURE__ */ jsxs(Button, {
                size: "sm",
                variant: "ghost",
                onClick: () => setLogsSvc(svc.name),
                children: [/* @__PURE__ */ jsx(ScrollText, {
                  className: "h-3 w-3"
                }), " logs"]
              })]
            })]
          }, svc.name);
        }), (data == null ? void 0 : data.length) === 0 && /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Sin servicios en el compose."
        })]
      })]
    }), logsSvc && /* @__PURE__ */ jsx(LogsDialog, {
      service: logsSvc,
      open: true,
      onOpenChange: (next) => {
        if (!next) setLogsSvc(null);
      }
    })]
  });
});
const route6 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: services
}, Symbol.toStringTag, { value: "Module" }));
const Label = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  LabelPrimitive.Root,
  {
    ref,
    className: cn(
      "text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70",
      className
    ),
    ...props
  }
));
Label.displayName = LabelPrimitive.Root.displayName;
const logs = UNSAFE_withComponentProps(function LogsRoute() {
  const {
    data: services2
  } = useQuery({
    queryKey: ["services"],
    queryFn: () => api("/api/services")
  });
  const [service, setService] = useState("");
  const [follow, setFollow] = useState(true);
  const [tail, setTail] = useState(200);
  const [lines, setLines] = useState([]);
  const preRef = useRef(null);
  const url = service ? `/api/logs/${encodeURIComponent(service)}?tail=${tail}&follow=${follow ? 1 : 0}` : null;
  useEventSource(url, (line) => {
    setLines((prev) => {
      const next = [...prev, line];
      const trimmed = next.length > 2e3 ? next.slice(next.length - 2e3) : next;
      queueMicrotask(() => {
        if (preRef.current) {
          preRef.current.scrollTop = preRef.current.scrollHeight;
        }
      });
      return trimmed;
    });
  });
  function download() {
    const blob = new Blob([lines.join("\n")], {
      type: "text/plain"
    });
    const u = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = u;
    a.download = `${service || "service"}.log`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(u);
  }
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsxs("div", {
      children: [/* @__PURE__ */ jsx("h1", {
        className: "text-2xl font-semibold tracking-tight",
        children: "Logs en vivo"
      }), /* @__PURE__ */ jsxs("p", {
        className: "text-sm text-muted-foreground",
        children: ["Stream SSE de ", /* @__PURE__ */ jsx("code", {
          children: "docker logs --tail N -f"
        }), "."]
      })]
    }), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsxs(CardTitle, {
          className: "flex items-center gap-2 text-base",
          children: [/* @__PURE__ */ jsx(Terminal, {
            className: "h-4 w-4"
          }), "Stream"]
        }), /* @__PURE__ */ jsx(CardDescription, {
          children: "Selecciona un servicio y (opcional) síguelo."
        })]
      }), /* @__PURE__ */ jsxs(CardContent, {
        className: "space-y-4",
        children: [/* @__PURE__ */ jsxs("div", {
          className: "flex flex-wrap items-end gap-3",
          children: [/* @__PURE__ */ jsxs("div", {
            className: "flex flex-col gap-1",
            children: [/* @__PURE__ */ jsx(Label, {
              htmlFor: "svc",
              children: "Servicio"
            }), /* @__PURE__ */ jsxs("select", {
              id: "svc",
              value: service,
              onChange: (e) => setService(e.target.value),
              className: "h-9 rounded-md border border-border bg-background px-2 text-sm",
              children: [/* @__PURE__ */ jsx("option", {
                value: "",
                children: "— elige —"
              }), services2 == null ? void 0 : services2.map((s) => /* @__PURE__ */ jsx("option", {
                value: s.name,
                children: s.name
              }, s.name))]
            })]
          }), /* @__PURE__ */ jsxs("div", {
            className: "flex flex-col gap-1",
            children: [/* @__PURE__ */ jsx(Label, {
              htmlFor: "tail",
              children: "tail"
            }), /* @__PURE__ */ jsx(Input, {
              id: "tail",
              type: "number",
              min: 10,
              max: 5e3,
              value: tail,
              onChange: (e) => setTail(Number(e.target.value) || 200),
              className: "w-24"
            })]
          }), /* @__PURE__ */ jsxs("div", {
            className: "flex items-center gap-2 pb-1",
            children: [/* @__PURE__ */ jsx(Switch, {
              id: "follow",
              checked: follow,
              onCheckedChange: setFollow
            }), /* @__PURE__ */ jsx(Label, {
              htmlFor: "follow",
              children: "follow"
            })]
          }), /* @__PURE__ */ jsxs("div", {
            className: "ml-auto flex gap-2",
            children: [/* @__PURE__ */ jsxs(Button, {
              variant: "outline",
              onClick: () => setLines([]),
              children: [/* @__PURE__ */ jsx(Trash2, {
                className: "h-4 w-4"
              }), " Clear"]
            }), /* @__PURE__ */ jsxs(Button, {
              variant: "outline",
              onClick: download,
              disabled: lines.length === 0,
              children: [/* @__PURE__ */ jsx(Download, {
                className: "h-4 w-4"
              }), " Descargar"]
            })]
          })]
        }), /* @__PURE__ */ jsx("pre", {
          ref: preRef,
          className: "max-h-[60vh] overflow-auto rounded-md border border-border bg-background/60 p-3 font-mono text-xs whitespace-pre-wrap",
          children: lines.length === 0 ? service ? "Esperando salida…" : "Selecciona un servicio para empezar." : lines.join("\n")
        })]
      })]
    })]
  });
});
const route7 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: logs
}, Symbol.toStringTag, { value: "Module" }));
const Tabs = TabsPrimitive.Root;
const TabsList = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  TabsPrimitive.List,
  {
    ref,
    className: cn(
      "inline-flex h-9 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground",
      className
    ),
    ...props
  }
));
TabsList.displayName = TabsPrimitive.List.displayName;
const TabsTrigger = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  TabsPrimitive.Trigger,
  {
    ref,
    className: cn(
      "inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow",
      className
    ),
    ...props
  }
));
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName;
const TabsContent = React.forwardRef(({ className, ...props }, ref) => /* @__PURE__ */ jsx(
  TabsPrimitive.Content,
  {
    ref,
    className: cn(
      "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
      className
    ),
    ...props
  }
));
TabsContent.displayName = TabsPrimitive.Content.displayName;
const ENV_KEYS = ["HOST_HOME", "HOST_UID", "HOST_GID", "MCP_TOOLS_ROOT", "MCP_TOOLS_DATA", "MCP_TOOLS_BIND", "MEM0_USER_ID"];
const MEM0_KEYS = ["MEM0_PROVIDER", "MEM0_LLM_MODEL", "MEM0_EMBED_PROVIDER", "MEM0_EMBED_MODEL", "MEM0_OLLAMA_URL", "MEM0_QDRANT_URL", "MEM0_COLLECTION", "MEM0_ENABLE_GRAPH", "MEM0_HISTORY_DB_PATH", "MEM0_OLLAMA_THINK"];
function EnvTable({
  title,
  keys,
  data,
  onSubmit
}) {
  const initial = {};
  for (const k2 of keys) initial[k2] = (data == null ? void 0 : data[k2]) ?? "";
  const [values, setValues] = useState(initial);
  const [busy, setBusy] = useState(false);
  if (data && !busy && JSON.stringify(values) !== JSON.stringify(initial) && Object.values(values).every((v) => v === "" || data[k])) ;
  async function submit() {
    setBusy(true);
    try {
      await onSubmit(values);
      toast.success(`${title} guardado`);
    } catch (err) {
      toast.error(`No se pudo guardar ${title}`, {
        description: err instanceof Error ? err.message : String(err)
      });
    } finally {
      setBusy(false);
    }
  }
  return /* @__PURE__ */ jsxs("div", {
    className: "grid gap-3",
    children: [keys.map((k2) => /* @__PURE__ */ jsxs("div", {
      className: "grid gap-1",
      children: [/* @__PURE__ */ jsx(Label, {
        htmlFor: k2,
        children: k2
      }), /* @__PURE__ */ jsx(Input, {
        id: k2,
        value: values[k2] ?? "",
        onChange: (e) => setValues({
          ...values,
          [k2]: e.target.value
        })
      })]
    }, k2)), /* @__PURE__ */ jsxs(Button, {
      onClick: submit,
      disabled: busy,
      className: "justify-self-start",
      children: [/* @__PURE__ */ jsx(Save, {
        className: "h-4 w-4"
      }), " Guardar ", title]
    })]
  });
}
const settings = UNSAFE_withComponentProps(function SettingsRoute() {
  const qc = useQueryClient();
  const {
    data
  } = useQuery({
    queryKey: ["status"],
    queryFn: () => api("/api/status")
  });
  const syncMut = useMutation({
    mutationFn: (path) => api(path, {
      method: "POST"
    }),
    onSuccess: (res, path) => {
      toast.success(`sync ${path}`, {
        description: `job ${res.job_id}`
      });
      qc.invalidateQueries({
        queryKey: ["status"]
      });
    }
  });
  return /* @__PURE__ */ jsxs("div", {
    className: "space-y-6",
    children: [/* @__PURE__ */ jsx("div", {
      className: "flex items-end justify-between",
      children: /* @__PURE__ */ jsxs("div", {
        className: "space-y-1",
        children: [/* @__PURE__ */ jsx("h1", {
          className: "text-2xl font-semibold tracking-tight",
          children: "Settings"
        }), /* @__PURE__ */ jsx("p", {
          className: "text-sm text-muted-foreground",
          children: "Edita los archivos .env y lanza los sincronizadores."
        })]
      })
    }), /* @__PURE__ */ jsxs(Tabs, {
      defaultValue: "env",
      children: [/* @__PURE__ */ jsxs(TabsList, {
        children: [/* @__PURE__ */ jsx(TabsTrigger, {
          value: "env",
          children: ".env"
        }), /* @__PURE__ */ jsx(TabsTrigger, {
          value: "mem0",
          children: ".env.mem0"
        })]
      }), /* @__PURE__ */ jsx(TabsContent, {
        value: "env",
        children: /* @__PURE__ */ jsxs(Card, {
          children: [/* @__PURE__ */ jsxs(CardHeader, {
            children: [/* @__PURE__ */ jsx(CardTitle, {
              className: "text-base",
              children: "Variables de entorno"
            }), /* @__PURE__ */ jsx(CardDescription, {
              children: "HOST_*, MCP_TOOLS_*, MEM0_USER_ID."
            })]
          }), /* @__PURE__ */ jsx(CardContent, {
            children: /* @__PURE__ */ jsx(EnvTable, {
              title: ".env",
              keys: ENV_KEYS,
              data: data == null ? void 0 : data.env,
              onSubmit: async (values) => {
                const res = await api("/api/env", {
                  method: "POST",
                  body: {
                    values
                  }
                });
                qc.invalidateQueries({
                  queryKey: ["status"]
                });
                return res;
              }
            })
          })]
        })
      }), /* @__PURE__ */ jsx(TabsContent, {
        value: "mem0",
        children: /* @__PURE__ */ jsxs(Card, {
          children: [/* @__PURE__ */ jsxs(CardHeader, {
            children: [/* @__PURE__ */ jsx(CardTitle, {
              className: "text-base",
              children: "mem0"
            }), /* @__PURE__ */ jsx(CardDescription, {
              children: "Proveedor, modelos, paths."
            })]
          }), /* @__PURE__ */ jsx(CardContent, {
            children: /* @__PURE__ */ jsx(EnvTable, {
              title: ".env.mem0",
              keys: MEM0_KEYS,
              data: data == null ? void 0 : data.env_mem0,
              onSubmit: async (values) => {
                const res = await api("/api/env-mem0", {
                  method: "POST",
                  body: {
                    values
                  }
                });
                qc.invalidateQueries({
                  queryKey: ["status"]
                });
                return res;
              }
            })
          })]
        })
      })]
    }), /* @__PURE__ */ jsx(Separator, {}), /* @__PURE__ */ jsxs(Card, {
      children: [/* @__PURE__ */ jsxs(CardHeader, {
        children: [/* @__PURE__ */ jsxs(CardTitle, {
          className: "flex items-center gap-2 text-base",
          children: [/* @__PURE__ */ jsx(Settings, {
            className: "h-4 w-4"
          }), "Sincronizadores"]
        }), /* @__PURE__ */ jsx(CardDescription, {
          children: "Re-registra MCPs, skills o reglas en los clientes soportados."
        })]
      }), /* @__PURE__ */ jsxs(CardContent, {
        className: "flex flex-wrap gap-2",
        children: [/* @__PURE__ */ jsxs(Button, {
          variant: "outline",
          disabled: syncMut.isPending,
          onClick: () => syncMut.mutate("/api/mcp-config/sync"),
          children: [/* @__PURE__ */ jsx(RefreshCcw, {
            className: "h-4 w-4"
          }), " Re-run mcp-config"]
        }), /* @__PURE__ */ jsxs(Button, {
          variant: "outline",
          disabled: syncMut.isPending,
          onClick: () => syncMut.mutate("/api/skills/sync"),
          children: [/* @__PURE__ */ jsx(RefreshCcw, {
            className: "h-4 w-4"
          }), " Sync skills"]
        }), /* @__PURE__ */ jsxs(Button, {
          variant: "outline",
          disabled: syncMut.isPending,
          onClick: () => syncMut.mutate("/api/rules/sync"),
          children: [/* @__PURE__ */ jsx(RefreshCcw, {
            className: "h-4 w-4"
          }), " Sync rules"]
        })]
      })]
    })]
  });
});
const route8 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: settings
}, Symbol.toStringTag, { value: "Module" }));
const setup = UNSAFE_withComponentProps(function SetupRoute() {
  const nav = useNavigate();
  const [token, setLocal] = useState("");
  const [err, setErr] = useState(null);
  function submit(e) {
    e.preventDefault();
    const t = token.trim();
    if (t.length < 8) {
      setErr("El token parece demasiado corto. Cópialo del output de `mcp-tools install`.");
      return;
    }
    setToken(t);
    nav("/", {
      replace: true
    });
  }
  return /* @__PURE__ */ jsx("div", {
    className: "flex min-h-[60vh] items-center justify-center",
    children: /* @__PURE__ */ jsx(motion.div, {
      initial: {
        opacity: 0,
        y: 8
      },
      animate: {
        opacity: 1,
        y: 0
      },
      transition: {
        duration: 0.3,
        ease: "easeOut"
      },
      className: "w-full max-w-md",
      children: /* @__PURE__ */ jsxs(Card, {
        children: [/* @__PURE__ */ jsxs(CardHeader, {
          children: [/* @__PURE__ */ jsxs(CardTitle, {
            className: "flex items-center gap-2",
            children: [/* @__PURE__ */ jsx(KeyRound, {
              className: "h-5 w-5"
            }), "Configuración inicial"]
          }), /* @__PURE__ */ jsxs(CardDescription, {
            children: ["Pega el token que imprimió ", /* @__PURE__ */ jsx("code", {
              children: "mcp-tools install"
            }), ". Se guarda en este navegador (localStorage)."]
          })]
        }), /* @__PURE__ */ jsx(CardContent, {
          children: /* @__PURE__ */ jsxs("form", {
            className: "space-y-3",
            onSubmit: submit,
            children: [/* @__PURE__ */ jsxs("div", {
              className: "grid gap-1",
              children: [/* @__PURE__ */ jsx(Label, {
                htmlFor: "token",
                children: "Bearer token"
              }), /* @__PURE__ */ jsx(Input, {
                id: "token",
                autoFocus: true,
                placeholder: "abcdef0123…",
                value: token,
                onChange: (e) => setLocal(e.target.value)
              })]
            }), err && /* @__PURE__ */ jsxs(Alert, {
              variant: "destructive",
              children: [/* @__PURE__ */ jsx(AlertTitle, {
                children: "Token inválido"
              }), /* @__PURE__ */ jsx(AlertDescription, {
                children: err
              })]
            }), /* @__PURE__ */ jsx(Button, {
              type: "submit",
              className: "w-full",
              children: "Guardar y entrar"
            })]
          })
        })]
      })
    })
  });
});
const route9 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: setup
}, Symbol.toStringTag, { value: "Module" }));
const notFound = UNSAFE_withComponentProps(function NotFoundRoute() {
  return /* @__PURE__ */ jsx("div", {
    className: "flex min-h-[60vh] items-center justify-center",
    children: /* @__PURE__ */ jsxs(motion.div, {
      initial: {
        opacity: 0,
        y: 8
      },
      animate: {
        opacity: 1,
        y: 0
      },
      className: "text-center",
      children: [/* @__PURE__ */ jsx("p", {
        className: "font-mono text-sm text-muted-foreground",
        children: "404"
      }), /* @__PURE__ */ jsx("h1", {
        className: "mt-2 text-3xl font-semibold",
        children: "Página no encontrada"
      }), /* @__PURE__ */ jsx(Link, {
        to: "/",
        className: "mt-4 inline-block text-sm text-foreground underline-offset-4 hover:underline",
        children: "Volver al dashboard"
      })]
    })
  });
});
const route10 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: notFound
}, Symbol.toStringTag, { value: "Module" }));
const serverManifest = { "entry": { "module": "/assets/entry.client-CFYeYJb2.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/index-D6-bsNtU.js"], "css": [] }, "routes": { "root": { "id": "root", "parentId": void 0, "path": "", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": true, "module": "/assets/root-Bu9LT1sV.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/index-D6-bsNtU.js", "/assets/index-BCRUn3uB.js", "/assets/mutation-8rnjaNmE.js", "/assets/QueryClientProvider-D1DQw-Y_.js"], "css": ["/assets/root-BGn3ARXi.css"], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/shell": { "id": "routes/shell", "parentId": "root", "path": void 0, "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/shell-BLrtQKjX.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/react-rdaJu49V.js", "/assets/index-BCRUn3uB.js", "/assets/index-Bl__oYkv.js", "/assets/index-zZZvJZEZ.js", "/assets/badge-DNF9kdEw.js", "/assets/separator-CJhKR8n6.js", "/assets/layers-DjDZBNiq.js", "/assets/database--8UUv_8R.js", "/assets/cog-DogRb3Ih.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/index-D6-bsNtU.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/_index": { "id": "routes/_index", "parentId": "routes/shell", "path": void 0, "index": true, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/_index-YP_83Rxr.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/react-rdaJu49V.js", "/assets/index-Bl__oYkv.js", "/assets/card-szXRj7eb.js", "/assets/badge-DNF9kdEw.js", "/assets/QueryClientProvider-D1DQw-Y_.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/tools": { "id": "routes/tools", "parentId": "routes/shell", "path": "tools", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/tools-DKqy0kXp.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/useMutation-C31d_FdS.js", "/assets/react-rdaJu49V.js", "/assets/mutation-8rnjaNmE.js", "/assets/index-Bl__oYkv.js", "/assets/sse-Cz8YVRRb.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/badge-DNF9kdEw.js", "/assets/switch-CLke56Bj.js", "/assets/dialog-B90JZ3cw.js", "/assets/alert-BurBd9IO.js", "/assets/loader-circle-CZSQBVsi.js", "/assets/trash-2-CMWBnehK.js", "/assets/refresh-ccw-CnbupJWH.js", "/assets/index-D6-bsNtU.js", "/assets/index-QRKxWi5e.js", "/assets/index-BN9uEBb8.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/configure": { "id": "routes/configure", "parentId": "routes/shell", "path": "configure", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/configure-CKaDDy_L.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/useMutation-C31d_FdS.js", "/assets/react-rdaJu49V.js", "/assets/mutation-8rnjaNmE.js", "/assets/index-Bl__oYkv.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/switch-CLke56Bj.js", "/assets/badge-DNF9kdEw.js", "/assets/alert-BurBd9IO.js", "/assets/loader-circle-CZSQBVsi.js", "/assets/save-6IQ6iMUe.js", "/assets/layers-DjDZBNiq.js", "/assets/index-D6-bsNtU.js", "/assets/index-QRKxWi5e.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/models": { "id": "routes/models", "parentId": "routes/shell", "path": "models", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/models-Bbjrvp0n.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/useMutation-C31d_FdS.js", "/assets/react-rdaJu49V.js", "/assets/mutation-8rnjaNmE.js", "/assets/index-Bl__oYkv.js", "/assets/sse-Cz8YVRRb.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/badge-DNF9kdEw.js", "/assets/input-ECPHgTg6.js", "/assets/alert-BurBd9IO.js", "/assets/dialog-B90JZ3cw.js", "/assets/loader-circle-CZSQBVsi.js", "/assets/trash-2-CMWBnehK.js", "/assets/database--8UUv_8R.js", "/assets/index-D6-bsNtU.js", "/assets/index-QRKxWi5e.js", "/assets/index-BN9uEBb8.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/services": { "id": "routes/services", "parentId": "routes/shell", "path": "services", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/services-D8xCi9lv.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/useMutation-C31d_FdS.js", "/assets/react-rdaJu49V.js", "/assets/mutation-8rnjaNmE.js", "/assets/index-Bl__oYkv.js", "/assets/sse-Cz8YVRRb.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/badge-DNF9kdEw.js", "/assets/dialog-B90JZ3cw.js", "/assets/cog-DogRb3Ih.js", "/assets/loader-circle-CZSQBVsi.js", "/assets/index-D6-bsNtU.js", "/assets/index-QRKxWi5e.js", "/assets/index-BN9uEBb8.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/logs": { "id": "routes/logs", "parentId": "routes/shell", "path": "logs", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/logs-bmwIyzz4.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/sse-Cz8YVRRb.js", "/assets/index-Bl__oYkv.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/switch-CLke56Bj.js", "/assets/label-B-cHEqJO.js", "/assets/input-ECPHgTg6.js", "/assets/trash-2-CMWBnehK.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/index-D6-bsNtU.js", "/assets/index-QRKxWi5e.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/settings": { "id": "routes/settings", "parentId": "routes/shell", "path": "settings", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/settings-DkcEKOdp.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/useQuery-C1GwkDEO.js", "/assets/QueryClientProvider-D1DQw-Y_.js", "/assets/useMutation-C31d_FdS.js", "/assets/mutation-8rnjaNmE.js", "/assets/index-Bl__oYkv.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/input-ECPHgTg6.js", "/assets/label-B-cHEqJO.js", "/assets/separator-CJhKR8n6.js", "/assets/index-QRKxWi5e.js", "/assets/index-BN9uEBb8.js", "/assets/refresh-ccw-CnbupJWH.js", "/assets/save-6IQ6iMUe.js", "/assets/index-D6-bsNtU.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/setup": { "id": "routes/setup", "parentId": "routes/shell", "path": "setup", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/setup-BecAiRyE.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/react-rdaJu49V.js", "/assets/index-Bl__oYkv.js", "/assets/card-szXRj7eb.js", "/assets/index-zZZvJZEZ.js", "/assets/input-ECPHgTg6.js", "/assets/label-B-cHEqJO.js", "/assets/alert-BurBd9IO.js", "/assets/index-D6-bsNtU.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 }, "routes/not-found": { "id": "routes/not-found", "parentId": "root", "path": "*", "index": void 0, "caseSensitive": void 0, "hasAction": false, "hasLoader": false, "hasClientAction": false, "hasClientLoader": false, "hasClientMiddleware": false, "hasDefaultExport": true, "hasErrorBoundary": false, "module": "/assets/not-found-BiKZq45M.js", "imports": ["/assets/chunk-KS7C4IRE-cccgDKe1.js", "/assets/react-rdaJu49V.js"], "css": [], "clientActionModule": void 0, "clientLoaderModule": void 0, "clientMiddlewareModule": void 0, "hydrateFallbackModule": void 0 } }, "url": "/assets/manifest-1f0743c5.js", "version": "1f0743c5", "sri": void 0 };
const assetsBuildDirectory = "build/client";
const basename = "/";
const future = { "unstable_optimizeDeps": false, "v8_passThroughRequests": false, "v8_trailingSlashAwareDataRequests": false, "unstable_previewServerPrerendering": false, "v8_middleware": false, "v8_splitRouteModules": false, "v8_viteEnvironmentApi": false };
const ssr = true;
const isSpaMode = false;
const prerender = [];
const routeDiscovery = { "mode": "lazy", "manifestPath": "/__manifest" };
const publicPath = "/";
const entry = { module: entryServer };
const routes = {
  "root": {
    id: "root",
    parentId: void 0,
    path: "",
    index: void 0,
    caseSensitive: void 0,
    module: route0
  },
  "routes/shell": {
    id: "routes/shell",
    parentId: "root",
    path: void 0,
    index: void 0,
    caseSensitive: void 0,
    module: route1
  },
  "routes/_index": {
    id: "routes/_index",
    parentId: "routes/shell",
    path: void 0,
    index: true,
    caseSensitive: void 0,
    module: route2
  },
  "routes/tools": {
    id: "routes/tools",
    parentId: "routes/shell",
    path: "tools",
    index: void 0,
    caseSensitive: void 0,
    module: route3
  },
  "routes/configure": {
    id: "routes/configure",
    parentId: "routes/shell",
    path: "configure",
    index: void 0,
    caseSensitive: void 0,
    module: route4
  },
  "routes/models": {
    id: "routes/models",
    parentId: "routes/shell",
    path: "models",
    index: void 0,
    caseSensitive: void 0,
    module: route5
  },
  "routes/services": {
    id: "routes/services",
    parentId: "routes/shell",
    path: "services",
    index: void 0,
    caseSensitive: void 0,
    module: route6
  },
  "routes/logs": {
    id: "routes/logs",
    parentId: "routes/shell",
    path: "logs",
    index: void 0,
    caseSensitive: void 0,
    module: route7
  },
  "routes/settings": {
    id: "routes/settings",
    parentId: "routes/shell",
    path: "settings",
    index: void 0,
    caseSensitive: void 0,
    module: route8
  },
  "routes/setup": {
    id: "routes/setup",
    parentId: "routes/shell",
    path: "setup",
    index: void 0,
    caseSensitive: void 0,
    module: route9
  },
  "routes/not-found": {
    id: "routes/not-found",
    parentId: "root",
    path: "*",
    index: void 0,
    caseSensitive: void 0,
    module: route10
  }
};
const allowedActionOrigins = false;
export {
  allowedActionOrigins,
  serverManifest as assets,
  assetsBuildDirectory,
  basename,
  entry,
  future,
  isSpaMode,
  prerender,
  publicPath,
  routeDiscovery,
  routes,
  ssr
};
