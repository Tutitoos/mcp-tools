import { useEffect, useState, useCallback, useRef } from "react";
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
 * produced by internal/web/job.go's handleJobEvents.
 */
export function useJobStream(jobId: string | null) {
  const [state, setState] = useState<JobState>({
    lines: [],
    done: false,
    ok: false,
    error: null,
    open: false,
  });
  const pending = useRef<JobLine[]>([]);
  const flushScheduled = useRef(false);
  const rafId = useRef<number | null>(null);

  useEffect(() => {
    if (!jobId) return;
    let cancelled = false;
    let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
    let buf = "";
    const ac = new AbortController();
    pending.current = [];
    flushScheduled.current = false;

    function flushPending() {
      if (pending.current.length === 0) return;
      const toFlush = pending.current;
      pending.current = [];
      setState((s) => ({ ...s, lines: s.lines.concat(toFlush) }));
    }

    function queueLine(line: JobLine) {
      pending.current.push(line);
      if (!flushScheduled.current) {
        flushScheduled.current = true;
        rafId.current = requestAnimationFrame(() => {
          flushScheduled.current = false;
          rafId.current = null;
          flushPending();
        });
      }
    }

    (async () => {
      try {
        const res = await apiStream(`/api/jobs/${jobId}/events`, {
          signal: ac.signal,
        });
        if (!res.ok || !res.body) {
          flushPending();
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
        flushPending();
        setState((s) => ({ ...s, open: false }));
      } catch (err) {
        flushPending();
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
      if (event === "hello") return; // handshake frame (see internal/web/job.go handleJobEvents) — not a log line
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
        flushPending();
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
        queueLine({ stream: "system", text: data });
      } else {
        const stream = data.slice(0, sp);
        const text = data.slice(sp + 1);
        if (stream === "stdout" || stream === "stderr" || stream === "system") {
          queueLine({ stream, text });
        }
      }
    }

    return () => {
      cancelled = true;
      ac.abort();
      if (reader) {
        reader.cancel().catch(() => undefined);
      }
      if (rafId.current !== null) {
        cancelAnimationFrame(rafId.current);
        rafId.current = null;
      }
      pending.current = [];
      flushScheduled.current = false;
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
  onMessage: (evt: JobLine) => void,
) {
  const cbRef = useRef(onMessage);
  useEffect(() => {
    cbRef.current = onMessage;
  }, [onMessage]);
  useEffect(() => {
    if (!url) return;
    let cancelled = false;
    let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
    let buf = "";
    const ac = new AbortController();
    const pending: JobLine[] = [];
    let flushScheduled = false;
    let rafId: number | null = null;

    function flushPending() {
      if (pending.length === 0) return;
      const toFlush = pending.splice(0, pending.length);
      for (const line of toFlush) cbRef.current(line);
    }

    function queueLine(line: JobLine) {
      pending.push(line);
      if (!flushScheduled) {
        flushScheduled = true;
        rafId = requestAnimationFrame(() => {
          flushScheduled = false;
          rafId = null;
          flushPending();
        });
      }
    }

    (async () => {
      try {
        const res = await apiStream(url, { signal: ac.signal });
        if (!res.ok || !res.body) {
          cbRef.current({ stream: "system", text: "[error] stream handshake failed" });
          return;
        }
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
              if (!line.startsWith("data:")) continue;
              const data = line.slice(5).trim();
              if (!data) continue;
              const sp = data.indexOf(" ");
              if (sp === -1) {
                queueLine({ stream: "system", text: data });
                continue;
              }
              const stream = data.slice(0, sp);
              const text = data.slice(sp + 1);
              if (stream === "stdout" || stream === "stderr" || stream === "system") {
                queueLine({ stream, text });
              } else {
                // frame sintético sin canal (p.ej. "data: docker: ..." de handleLogsStream error path)
                queueLine({ stream: "system", text: data });
              }
            }
          }
        }
        flushPending();
      } catch {
        // connection torn down — typical when the tab closes
      }
    })();
    return () => {
      cancelled = true;
      ac.abort();
      if (reader) reader.cancel().catch(() => undefined);
      if (rafId !== null) cancelAnimationFrame(rafId);
      pending.length = 0;
      flushScheduled = false;
    };
  }, [url]);
}
