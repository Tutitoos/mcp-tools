import { useEffect, useRef, useState, useCallback } from "react";
import { apiStream } from "~/lib/api";

export type JobLine = {
  stream: "stdout" | "stderr" | "system";
  text: string;
};

export type JobDone = { ok: boolean; error?: string };

export type JobState = {
  lines: JobLine[];
  done: boolean;
  ok: boolean;
  error: string | null;
  open: boolean;
};

/**
 * Subscribe to a Server-Sent-Events stream from the API.
 * Parses the bare "<stream> <line>\n\n" and "event: done {json}" format
 * produced by internal/web/logstream.go.
 */
export function useJobStream(jobId: string | null) {
  const [state, setState] = useState<JobState>({
    lines: [],
    done: false,
    ok: false,
    error: null,
    open: false,
  });
  const closeRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!jobId) return;
    let cancelled = false;
    let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
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
            open: false,
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
          let idx: number;
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
          open: false,
        }));
      }
    })();

    function handleFrame(frame: string) {
      let event = "message";
      const dataLines: string[] = [];
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
        let error: string | undefined;
        try {
          const parsed = JSON.parse(data) as { ok: boolean; error?: string };
          ok = !!parsed.ok;
          error = parsed.error;
        } catch {
          // ignore malformed
        }
        setState((s) => ({
          ...s,
          done: true,
          ok,
          error: error ?? (ok ? null : s.error),
          open: false,
        }));
        return;
      }
      // Default frame: "<stream> <line>"
      if (!data) return;
      const sp = data.indexOf(" ");
      if (sp === -1) {
        setState((s) => ({
          ...s,
          lines: [...s.lines, { stream: "system", text: data }],
        }));
      } else {
        const stream = data.slice(0, sp);
        const text = data.slice(sp + 1);
        if (stream === "stdout" || stream === "stderr" || stream === "system") {
          setState((s) => ({
            ...s,
            lines: [...s.lines, { stream, text }],
          }));
        }
      }
    }

    return () => {
      cancelled = true;
      if (reader) {
        reader.cancel().catch(() => undefined);
      }
    };
  }, [jobId]);

  const reset = useCallback(() => {
    setState({ lines: [], done: false, ok: false, error: null, open: false });
  }, []);

  return { ...state, reset };
}

/**
 * Subscribe to a generic SSE stream (e.g. /api/logs/<service>). Each
 * message is the full line emitted by the server; no event/done framing.
 */
export function useEventSource(
  url: string | null,
  onMessage: (line: string) => void,
) {
  useEffect(() => {
    if (!url) return;
    let cancelled = false;
    let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
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
          let idx: number;
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
        // connection torn down — typical when the tab closes
      }
    })();
    return () => {
      cancelled = true;
      if (reader) reader.cancel().catch(() => undefined);
    };
  }, [url, onMessage]);
}