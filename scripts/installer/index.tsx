#!/usr/bin/env bun
import React, { useEffect, useState } from "react";
import { render, Box, Text } from "ink";
import { Spinner, StatusMessage } from "@inkjs/ui";
import { execa } from "execa";
import path from "node:path";
import fs from "node:fs";
import os from "node:os";

const REPO_DIR = path.resolve(new URL(".", import.meta.url).pathname, "../..");
const HOME = os.homedir();

type Status = "pending" | "running" | "ok" | "fail";

interface Step {
  key: string;
  label: string;
  run: () => Promise<void>;
}

const sh = (cmd: string) =>
  execa("bash", ["-c", cmd], { cwd: REPO_DIR, stdio: "pipe" });

/** Narrow an unknown thrown value to a readable error message. Handles execa's
 *  `.stderr` field without an unchecked cast. */
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
    run: async () => {
      await sh("command -v docker");
      await sh("docker compose version");
    },
  },
  {
    key: "env",
    label: "Generar .env desde el host",
    run: async () => {
      await sh("./scripts/init-env.sh");
    },
  },
  {
    key: "mem0-src",
    label: "Verificar clon de mem0-mcp-selfhosted",
    run: async () => {
      const envText = fs.readFileSync(path.join(REPO_DIR, ".env"), "utf8");
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
    run: async () => {
      await sh("./scripts/build.sh");
    },
  },
  {
    key: "wrappers",
    label: "Instalar wrappers en ~/.local/bin/",
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
    label: "Instalar skills globales",
    run: async () => {
      await sh("./scripts/install-skills.sh");
    },
  },
  {
    key: "rules",
    label: "Instalar RULES.md globales",
    run: async () => {
      await sh("./scripts/install-rules.sh");
    },
  },
  {
    key: "mcp-config",
    label: "Configurar MCP en Claude Code / OpenCode / OMP",
    run: async () => {
      await sh("bun scripts/install-mcp.ts");
    },
  },
  {
    key: "up",
    label: "Arrancar contenedores (docker compose up -d)",
    run: async () => {
      await sh("./scripts/up.sh");
    },
  },
  {
    key: "smoke",
    label: "Smoke test de los 3 MCPs",
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

const App: React.FC = () => {
  const [states, setStates] = useState<Record<string, Status>>(
    Object.fromEntries(STEPS.map((s) => [s.key, "pending" as Status])),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [done, setDone] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    (async () => {
      for (const step of STEPS) {
        setStates((s) => ({ ...s, [step.key]: "running" }));
        try {
          await step.run();
          setStates((s) => ({ ...s, [step.key]: "ok" }));
        } catch (err: unknown) {
          setStates((s) => ({ ...s, [step.key]: "fail" }));
          setErrors((er) => ({ ...er, [step.key]: errorMessage(err) }));
          setFailed(true);
          setDone(true);
          return;
        }
      }
      setDone(true);
    })();
  }, []);

  useEffect(() => {
    if (done) setTimeout(() => process.exit(failed ? 1 : 0), 100);
  }, [done, failed]);

  return (
    <Box flexDirection="column">
      <Text bold>mcp-tools installer</Text>
      <Box marginTop={1} flexDirection="column">
        {STEPS.map((s) => {
          const st = states[s.key];
          if (st === "running") {
            return (
              <Box key={s.key}>
                <Spinner label={s.label} />
              </Box>
            );
          }
          const variant =
            st === "ok" ? "success" : st === "fail" ? "error" : "info";
          return (
            <StatusMessage key={s.key} variant={variant}>
              {s.label}
            </StatusMessage>
          );
        })}
      </Box>
      {failed && (
        <Box marginTop={1} flexDirection="column">
          <Text color="red">Falló:</Text>
          {Object.entries(errors).map(([k, v]) => (
            <Box key={k} flexDirection="column">
              <Text color="red" bold>
                [{k}]
              </Text>
              <Text>{v}</Text>
            </Box>
          ))}
        </Box>
      )}
      {done && !failed && (
        <Box marginTop={1}>
          <Text color="green">Todo listo. Reinicia tu cliente MCP.</Text>
        </Box>
      )}
    </Box>
  );
};

render(<App />);
