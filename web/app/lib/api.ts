// Typed fetch wrapper for the mcp-tools web API. The API is
// unauthenticated by design -- bind to 127.0.0.1 (or rely on firewall)
// to restrict access.

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
  } | null;
  env: Record<string, string>;
  env_mem0: Record<string, string>;
  compose_services: { name: string; state: string }[] | null;
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