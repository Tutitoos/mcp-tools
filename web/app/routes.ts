import {
  type RouteConfig,
  index,
  layout,
  route,
} from "@react-router/dev/routes";

export default [
  layout("routes/shell.tsx", [
    index("routes/_index.tsx"),
    route("tools", "routes/tools.tsx"),
    route("configure", "routes/configure.tsx"),
    route("models", "routes/models.tsx"),
    route("services", "routes/services.tsx"),
    route("logs", "routes/logs.tsx"),
    route("settings", "routes/settings.tsx"),
    route("setup", "routes/setup.tsx"),
  ]),
  route("*", "routes/not-found.tsx"),
] satisfies RouteConfig;