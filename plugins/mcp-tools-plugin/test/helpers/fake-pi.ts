import type { ToolCallEvent } from "@oh-my-pi/pi-coding-agent";

/**
 * Minimal `ExtensionAPI` stub for unit tests: only the surface guards and
 * extensions in this package actually use (`on`, `getAllTools`, `logger`,
 * `sendMessage`), plus an `emit` helper to drive registered handlers
 * directly without a real runtime.
 *
 * `getAllTools` returns `string[]`, matching the real
 * `@oh-my-pi/pi-coding-agent` `ExtensionAPI` -- bridged MCP tool names, not
 * `{ name: string }` objects.
 */

type Listener = (event: unknown, ctx: unknown) => unknown;

export interface FakePi {
  on(evt: string, fn: Listener): void;
  getAllTools(): string[];
  emit(evt: string, event: unknown, ctx?: unknown): Promise<unknown>;
  logger: { debug(): void; info(): void; warn(): void; error(): void };
  sendMessage(message: unknown): void;
  /** Messages passed to `sendMessage`, in call order. */
  sentMessages: unknown[];
}

export function createFakePi(opts: { tools?: string[] } = {}): FakePi {
  const listeners = new Map<string, Listener[]>();
  const sentMessages: unknown[] = [];

  return {
    on(evt, fn) {
      const arr = listeners.get(evt) ?? [];
      arr.push(fn);
      listeners.set(evt, arr);
    },
    getAllTools: () => opts.tools ?? [],
    async emit(evt, event, ctx = {}) {
      let last: unknown;
      for (const fn of listeners.get(evt) ?? []) {
        const result = await fn(event, ctx);
        if (result !== undefined) last = result;
      }
      return last;
    },
    logger: {
      debug() {},
      info() {},
      warn() {},
      error() {},
    },
    get sentMessages() {
      return sentMessages;
    },
    sendMessage(message) {
      sentMessages.push(message);
    },
  };
}

export interface FakeCtx {
  sessionManager: { getSessionId(): string };
  cwd: string;
}

/** A minimal `ExtensionContext` stub keyed by session id. */
export function fakeCtx(sessionId = "session-1", cwd = "/repo"): FakeCtx {
  return { sessionManager: { getSessionId: () => sessionId }, cwd };
}

/** Builds a minimal `ToolCallEvent` fixture for a guard's `check()`. */
export function toolCallEvent(toolName: string, input: Record<string, unknown>): ToolCallEvent {
  return { type: "tool_call", toolCallId: "test-call", toolName, input } as ToolCallEvent;
}
