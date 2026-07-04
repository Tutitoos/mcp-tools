#!/usr/bin/env bun
import React, { useEffect, useState } from "react";
import { render, Box, Text } from "ink";
import { Spinner } from "@inkjs/ui";
import { execa } from "execa";
import path from "node:path";
import fs from "node:fs";
import os from "node:os";

const REPO_DIR = path.resolve(new URL(".", import.meta.url).pathname, "../..");
const HOME = os.homedir();

type Status = "pending" | "running" | "ok" | "fail";
type Phase = "Preparación" | "Build" | "Instalación" | "Arranque";

interface Step {
  key: string;
  label: string;
  phase: Phase;
  run: () => Promise<void>;
}

const IS_DRY = process.argv.slice(2).some((a) => a === "--dry" || a === "--dry-run");

let currentStepKey = "";
const dryCommands: Array<{ stepKey: string; cmd: string }> = [];

const sh = (cmd: string) => {
  if (IS_DRY) {
    dryCommands.push({ stepKey: currentStepKey, cmd });
    return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
  }
  return execa("bash", ["-c", cmd], { cwd: REPO_DIR, stdio: "pipe" });
};

/** Narrow an unknown thrown value to a readable error message. */
function errorMessage(err: unknown): string {
  if (err instanceof Error) {
    if ("stderr" in err && typeof err.stderr === "string" && err.stderr.length > 0) {
      return err.stderr;
    }
    return err.message;
  }
  return String(err);
}


const STEPS: Step[] = [
  {
    key: "prereq",
    label: "Comprobar prerequisitos (docker + docker compose)",
    phase: "Preparación",
    run: async () => {
      await sh("command -v docker");
      await sh("docker compose version");
    },
  },
  {
    key: "env",
    label: "Generar .env desde el host",
    phase: "Preparación",
    run: async () => {
      await sh("./scripts/init-env.sh");
    },
  },
  {
    key: "mem0-src",
    label: "Verificar clon de mem0-mcp-selfhosted",
    phase: "Preparación",
    run: async () => {
      const envPath = path.join(REPO_DIR, ".env");
      if (!fs.existsSync(envPath)) {
        if (IS_DRY) return;
        throw new Error(`.env ausente en ${envPath} — el paso 'env' debería haberlo generado`);
      }
      const envText = fs.readFileSync(envPath, "utf8");
      const m = envText.match(/^MEM0_SRC_PATH=(.+)$/m);
      if (!m) throw new Error("MEM0_SRC_PATH ausente en .env");
      const p = m[1].replace(/^\$HOME/, HOME);
      if (!fs.existsSync(path.join(p, "pyproject.toml"))) {
        throw new Error(
          `MEM0_SRC_PATH no existe o no es un repo válido: ${p}\n` +
            `Clona: git clone https://github.com/elvismdev/mem0-mcp-selfhosted ${p}`,
        );
      }
    },
  },
  {
    key: "build",
    label: "docker compose build (puede tardar)",
    phase: "Build",
    run: async () => {
      await sh("./scripts/build.sh");
    },
  },
  {
    key: "wrappers",
    label: "Wrappers en ~/.local/bin/",
    phase: "Instalación",
    run: async () => {
      await sh(`mkdir -p "${HOME}/.local/bin"`);
      for (const w of ["codebase-memory", "mem0", "headroom"]) {
        await sh(
          `ln -snf "${REPO_DIR}/scripts/wrappers/mcp-tools-${w}-docker" "${HOME}/.local/bin/"`,
        );
      }
    },
  },
  {
    key: "skills",
    label: "Skills globales",
    phase: "Instalación",
    run: async () => {
      await sh("./scripts/install-skills.sh");
    },
  },
  {
    key: "rules",
    label: "RULES.md globales",
    phase: "Instalación",
    run: async () => {
      await sh("./scripts/install-rules.sh");
    },
  },
  {
    key: "mcp-config",
    label: "Registrar MCPs en Claude Code / OpenCode / OMP",
    phase: "Instalación",
    run: async () => {
      await sh("bun scripts/install-mcp.ts");
    },
  },
  {
    key: "up",
    label: "Arrancar contenedores (docker compose up -d)",
    phase: "Arranque",
    run: async () => {
      await sh("./scripts/up.sh");
    },
  },
  {
    key: "smoke",
    label: "Smoke test MCP handshake",
    phase: "Arranque",
    run: async () => {
      await sh(`"${HOME}/.local/bin/mcp-tools-codebase-memory-docker" --version`);
      await sh(
        `timeout 5 "${HOME}/.local/bin/mcp-tools-headroom-docker" --help >/dev/null 2>&1 || true`,
      );
      const init = `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"installer","version":"1"}}}'`;
      await sh(
        `${init} | timeout 15 "${HOME}/.local/bin/mcp-tools-mem0-docker" | grep -q '"serverInfo"'`,
      );
    },
  },
];

const PHASES: Phase[] = ["Preparación", "Build", "Instalación", "Arranque"];

const STATUS_GLYPH: Record<Status, string> = {
  pending: "○",
  running: "◐",
  ok: "✔",
  fail: "✘",
};

const STATUS_COLOR: Record<Status, string> = {
  pending: "gray",
  running: "cyan",
  ok: "green",
  fail: "red",
};

const App: React.FC = () => {
  const [states, setStates] = useState<Record<string, Status>>(
    Object.fromEntries(STEPS.map((s) => [s.key, "pending" as Status])),
  );
  const [durations, setDurations] = useState<Record<string, number>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [totalMs, setTotalMs] = useState(0);
  const [done, setDone] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    (async () => {
      const wallStart = Date.now();
      for (const step of STEPS) {
        currentStepKey = step.key;
        setStates((s) => ({ ...s, [step.key]: "running" }));
        const t0 = Date.now();
        try {
          await step.run();
          const dt = Date.now() - t0;
          setDurations((d) => ({ ...d, [step.key]: dt }));
          setStates((s) => ({ ...s, [step.key]: "ok" }));
        } catch (err: unknown) {
          const dt = Date.now() - t0;
          setDurations((d) => ({ ...d, [step.key]: dt }));
          setStates((s) => ({ ...s, [step.key]: "fail" }));
          setErrors((er) => ({ ...er, [step.key]: errorMessage(err) }));
          setFailed(true);
          setTotalMs(Date.now() - wallStart);
          setDone(true);
          return;
        }
      }
      setTotalMs(Date.now() - wallStart);
      setDone(true);
    })();
  }, []);

  useEffect(() => {
    if (done) setTimeout(() => process.exit(failed ? 1 : 0), 100);
  }, [done, failed]);

  const totalSteps = STEPS.length;

  return (
    <Box flexDirection="column">
      <Box flexDirection="column" marginBottom={1}>
        <Box>
          <Text bold color="magentaBright">mcp-tools</Text>
          <Text dimColor>  installer</Text>
        </Box>
        <Text dimColor>self-hosted MCP servers para Claude Code, OpenCode y OMP</Text>
      </Box>

      {IS_DRY && (
        <Box marginBottom={1}>
          <Text backgroundColor="yellow" color="black" bold> DRY RUN </Text>
          <Text dimColor>  no se ejecuta nada; solo se muestra qué haría</Text>
        </Box>
      )}

      {PHASES.map((phase) => {
        const stepsInPhase = STEPS.filter((s) => s.phase === phase);
        return (
          <Box
            key={phase}
            flexDirection="column"
            marginBottom={1}
            borderStyle="single"
            borderColor="cyan"
            borderTop={false}
            borderRight={false}
            borderBottom={false}
            paddingLeft={1}
          >
            <Text color="cyan" bold>{phase}</Text>
            {stepsInPhase.map((s) => {
              const st = states[s.key];
              const idx = STEPS.findIndex((x) => x.key === s.key) + 1;
              const dt = durations[s.key];
              if (st === "running") {
                return (
                  <Box key={s.key}>
                    <Text dimColor>{String(idx).padStart(2, "0")}  </Text>
                    <Spinner label={s.label} />
                  </Box>
                );
              }
              return (
                <Box key={s.key}>
                  <Text dimColor>{String(idx).padStart(2, "0")}  </Text>
                  <Text color={STATUS_COLOR[st]}>{STATUS_GLYPH[st]}  </Text>
                  <Text color={st === "fail" ? "red" : undefined}>{s.label.padEnd(52, " ")}</Text>
                  {typeof dt === "number" && (
                    <Text dimColor>{(dt / 1000).toFixed(1)}s</Text>
                  )}
                </Box>
              );
            })}
          </Box>
        );
      })}

      {failed && (
        <Box marginTop={1} flexDirection="column">
          <Text color="red" bold>
            Falló tras {(totalMs / 1000).toFixed(1)}s
          </Text>
          {Object.entries(errors).map(([k, v]) => (
            <Box key={k} flexDirection="column" marginTop={1}>
              <Text color="red" bold>
                [{k}]
              </Text>
              <Text>{v}</Text>
            </Box>
          ))}
          <Box marginTop={1}>
            <Text dimColor>
              Corrige el error y relanza `./install.sh` — el installer es idempotente.
            </Text>
          </Box>
        </Box>
      )}

      {done && !failed && IS_DRY && (
        <Box flexDirection="column" marginTop={1}>
          <Box>
            <Text backgroundColor="green" color="black" bold> DRY RUN OK </Text>
            <Text dimColor>  {(totalMs / 1000).toFixed(1)}s · {dryCommands.length} comandos en {STEPS.filter((s) => dryCommands.some((c) => c.stepKey === s.key)).length} pasos</Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            <Text dimColor>Comandos que ejecutaría (sh -c):</Text>
            {STEPS.map((s) => {
              const stepCmds = dryCommands.filter((c) => c.stepKey === s.key);
              if (stepCmds.length === 0) return null;
              const idx = STEPS.findIndex((x) => x.key === s.key) + 1;
              return (
                <Box
                  key={s.key}
                  flexDirection="column"
                  marginTop={1}
                  borderStyle="single"
                  borderColor="gray"
                  borderTop={false}
                  borderRight={false}
                  borderBottom={false}
                  paddingLeft={1}
                >
                  <Text>
                    <Text color="cyan" bold>{String(idx).padStart(2, "0")}</Text>
                    <Text dimColor>  </Text>
                    <Text>{s.label}</Text>
                  </Text>
                  {stepCmds.map((c, i) => (
                    <Text key={i} dimColor>   $ {c.cmd.split(HOME).join("~")}</Text>
                  ))}
                </Box>
              );
            })}
          </Box>
          <Box marginTop={1}>
            <Text dimColor>Relanza sin </Text>
            <Text color="yellow" bold>--dry</Text>
            <Text dimColor> para aplicar.</Text>
          </Box>
        </Box>
      )}

      {done && !failed && !IS_DRY && (
        <Box flexDirection="column" marginTop={1}>
          <Box>
            <Text backgroundColor="green" color="black" bold> INSTALADO </Text>
            <Text dimColor>  {(totalMs / 1000).toFixed(1)}s</Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            <Text dimColor>Próximos pasos:</Text>
            <Text>  → Reinicia tu cliente MCP (Claude Code, OpenCode, OMP).</Text>
            <Text>  → Verifica: <Text color="cyan" bold>claude mcp list</Text> · los 3 servers como ✔ Connected.</Text>
            <Text>  → Config avanzada: <Text color="cyan" bold>docs/ADVANCED.md</Text></Text>
          </Box>
        </Box>
      )}
    </Box>
  );
};

render(<App />);
