// Typed fetch wrapper for the mcp-tools web API.
//
// Reads the bearer token from localStorage (set on the /setup route) and
// attaches `Authorization: Bearer <token>` when present. Throws an `ApiError`
// with the response body on non-2xx so callers can surface server-side error
// messages in toasts/alerts.

const TOKEN_KEY = "mcp-tools-token";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

type Init = Omit<RequestInit, "body"> & { body?: unknown };

export async function api<T>(path: string, init: Init = {}): Promise<T> {
  const headers = new Headers(init.headers ?? {});
  headers.set("Accept", "application/json");
  let body: BodyInit | undefined;
  if (init.body !== undefined) {
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
  let parsed: unknown = text;
  if (text.length > 0) {
    try {
      parsed = JSON.parse(text);
    } catch {
      // keep text
    }
  }
  if (!res.ok) {
    const message =
      parsed && typeof parsed === "object" && "error" in parsed
        ? String((parsed as { error: unknown }).error)
        : res.statusText;
    throw new ApiError(res.status, message, parsed);
  }
  return parsed as T;
}

export async function apiStream(path: string): Promise<Response> {
  const headers = new Headers();
  headers.set("Accept", "text/event-stream");
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetch(path, { headers });
}

// ─── Typed view models ──────────────────────────────────────────────────

export type DeployKind = "Host" | "Docker" | "Sudo" | "?";

export type ToolView = {
  key: string;
  label: string;
  summary: string;
  deploy: DeployKind;
  default_on: boolean;
  deps: string[];
  installed: boolean;
  selected: boolean;
  version: string;
  extra: Record<string, unknown>;
};

export type StatusPayload = {
  state: {
    selected: string[];
    versions: Record<string, string>;
    updated_at: string;
  };
  env: Record<string, string>;
  env_mem0: Record<string, string>;
  compose_services: { name: string; state: string }[];
  docker_running: boolean;
};

export type ServiceView = { name: string; state: string };

export type ModelView = { tag: string; size: string; modified: string };

export type JobResponse = { ok: boolean; job_id: string };

export type VersionResponse = {
  version: string;
  commit: string;
  date: string;
};