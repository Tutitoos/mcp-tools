import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";

export interface SessionMap<T> {
  get(sessionId: string): T;
  delete(sessionId: string): void;
}

/**
 * Per-session state keyed by session id, auto-created via `factory()` on
 * first access. Registers cleanup on `session_shutdown` (process exit),
 * `session_switch`, and `session_branch` so state doesn't accumulate for
 * sessions the extension will never see again -- fixes the module-global
 * `Map` that never shrank in the original extensions.
 */
export function sessionMap<T>(pi: ExtensionAPI, factory: () => T): SessionMap<T> {
  const map = new Map<string, T>();
  const cleanup = (_event: unknown, ctx: ExtensionContext): void => {
    map.delete(ctx.sessionManager.getSessionId());
  };

  pi.on("session_shutdown", cleanup);
  pi.on("session_switch", cleanup);
  pi.on("session_branch", cleanup);

  return {
    get(sessionId) {
      let value = map.get(sessionId);
      if (value === undefined) {
        value = factory();
        map.set(sessionId, value);
      }
      return value;
    },
    delete(sessionId) {
      map.delete(sessionId);
    },
  };
}
