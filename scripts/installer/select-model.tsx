#!/usr/bin/env bun
import React, { useState } from "react";
import { render, Box, Text } from "ink";
import { Select, ConfirmInput, Spinner } from "@inkjs/ui";
import { execa } from "execa";
import fs from "node:fs";
import path from "node:path";

const REPO_DIR = path.resolve(new URL(".", import.meta.url).pathname, "../..");
const ENV_MEM0_PATH = path.join(REPO_DIR, ".env.mem0");

if (!fs.existsSync(ENV_MEM0_PATH)) {
  console.error(`ERROR: ${ENV_MEM0_PATH} no existe.`);
  console.error("Créalo primero desde el bloque del README (sección Configuración → .env.mem0).");
  process.exit(1);
}

interface ModelOption {
  label: string;
  value: string;
}

const LLM_MODELS: ModelOption[] = [
  { label: "qwen2.5:7b        7B    multilingüe, tool calling maduro (default)", value: "qwen2.5:7b" },
  { label: "qwen3:8b          8B    siguiente gen de qwen, mejor calidad", value: "qwen3:8b" },
  { label: "mistral-nemo:12b  12B   Mistral+NVIDIA, contexto 128k", value: "mistral-nemo:12b" },
  { label: "llama3.1:8b       8B    Meta, menos multilingüe que qwen", value: "llama3.1:8b" },
  { label: "mistral:7b        7B    function calling desde v0.3", value: "mistral:7b" },
  { label: "qwen3:4b          4B    compacto dentro de qwen3", value: "qwen3:4b" },
  { label: "qwen2.5:3b        3B    ligero dentro de qwen2.5", value: "qwen2.5:3b" },
  { label: "llama3.2:3b       3B    Meta ligero", value: "llama3.2:3b" },
  { label: "granite3.1-moe:3b 3B    IBM MoE, punchea por encima", value: "granite3.1-moe:3b" },
  { label: "smollm2:1.7b      1.7B  mínimo viable, solo probar", value: "smollm2:1.7b" },
];

const EMBED_MODELS: ModelOption[] = [
  { label: "bge-m3                  1024 dims, multilingüe 100+ (default)", value: "bge-m3" },
  { label: "mxbai-embed-large       mixedbread.ai (verificar dim con ollama show)", value: "mxbai-embed-large" },
  { label: "snowflake-arctic-embed  familia Snowflake, varias variantes", value: "snowflake-arctic-embed" },
  { label: "nomic-embed-text        contexto largo (verificar dim)", value: "nomic-embed-text" },
  { label: "all-minilm              mínimo (22m/33m params), solo pruebas", value: "all-minilm" },
];

const KIND_OPTIONS: ModelOption[] = [
  { label: "LLM     (MEM0_LLM_MODEL)   — el que extrae memorias", value: "llm" },
  { label: "Embed   (MEM0_EMBED_MODEL) — vectores en qdrant", value: "embed" },
];

function readEnvMem0(): Record<string, string> {
  const text = fs.readFileSync(ENV_MEM0_PATH, "utf8");
  const map: Record<string, string> = {};
  for (const line of text.split("\n")) {
    const m = line.match(/^([A-Z_][A-Z0-9_]*)=(.*)$/);
    if (m) map[m[1]] = m[2];
  }
  return map;
}

function updateEnvMem0(updates: Record<string, string>): void {
  let text = fs.readFileSync(ENV_MEM0_PATH, "utf8");
  for (const [k, v] of Object.entries(updates)) {
    const re = new RegExp(`^${k}=.*$`, "m");
    if (re.test(text)) {
      text = text.replace(re, `${k}=${v}`);
    } else {
      text += (text.endsWith("\n") ? "" : "\n") + `${k}=${v}\n`;
    }
  }
  fs.writeFileSync(ENV_MEM0_PATH, text);
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) {
    if ("stderr" in err && typeof err.stderr === "string" && err.stderr.length > 0) {
      return err.stderr;
    }
    return err.message;
  }
  return String(err);
}

type Kind = "llm" | "embed";
type Phase =
  | "chooseKind"
  | "chooseModel"
  | "confirm"
  | "pulling"
  | "restarting"
  | "done"
  | "error";

const App: React.FC = () => {
  const current = readEnvMem0();
  const [phase, setPhase] = useState<Phase>("chooseKind");
  const [kind, setKind] = useState<Kind | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const envVar = kind === "llm" ? "MEM0_LLM_MODEL" : "MEM0_EMBED_MODEL";
  const currentValue = current[envVar] ?? "";
  const needsThinkFlag =
    kind === "llm" &&
    selected !== null &&
    (selected.startsWith("qwen3") || selected.startsWith("deepseek-r1"));

  const runPipeline = async () => {
    if (!selected || !kind) return;
    try {
      setPhase("pulling");
      await execa("docker", ["exec", "mcp-tools-ollama", "ollama", "pull", selected], {
        stdio: "pipe",
      });

      const updates: Record<string, string> = { [envVar]: selected };
      if (needsThinkFlag) updates.MEM0_OLLAMA_THINK = "false";
      updateEnvMem0(updates);

      setPhase("restarting");
      await execa("docker", ["restart", "mcp-tools-mem0"], { stdio: "pipe" });

      setPhase("done");
      setTimeout(() => process.exit(0), 100);
    } catch (err: unknown) {
      setError(errorMessage(err));
      setPhase("error");
      setTimeout(() => process.exit(1), 100);
    }
  };

  return (
    <Box flexDirection="column">
      <Box flexDirection="column" marginBottom={1}>
        <Box>
          <Text bold color="magentaBright">mem0</Text>
          <Text dimColor>  cambiar modelo</Text>
        </Box>
        <Text dimColor>
          actual · LLM=<Text color="cyan">{current.MEM0_LLM_MODEL ?? "?"}</Text>
          <Text dimColor> · Embed=</Text>
          <Text color="cyan">{current.MEM0_EMBED_MODEL ?? "?"}</Text>
        </Text>
      </Box>

      {phase === "chooseKind" && (
        <Box flexDirection="column">
          <Text bold>¿Qué modelo cambiar?</Text>
          <Box marginTop={1}>
            <Select
              options={KIND_OPTIONS}
              onChange={(v) => {
                if (v === "llm" || v === "embed") {
                  setKind(v);
                  setPhase("chooseModel");
                }
              }}
            />
          </Box>
        </Box>
      )}

      {phase === "chooseModel" && kind && (
        <Box flexDirection="column">
          <Text bold>
            {kind === "llm" ? "LLM" : "Embeddings"} · actual: <Text color="cyan">{currentValue}</Text>
          </Text>
          <Text dimColor>
            {kind === "llm"
              ? "requisito: el tag debe llevar `tools` en https://ollama.com/library"
              : "aviso: si cambia la dim, hay que resetear la colección qdrant o cambiar MEM0_COLLECTION"}
          </Text>
          <Box marginTop={1}>
            <Select
              options={kind === "llm" ? LLM_MODELS : EMBED_MODELS}
              onChange={(v) => {
                setSelected(v);
                setPhase("confirm");
              }}
            />
          </Box>
        </Box>
      )}

      {phase === "confirm" && kind && selected && (
        <Box flexDirection="column">
          <Text>
            Selección: <Text color="cyan" bold>{selected}</Text>
          </Text>
          <Box marginTop={1} flexDirection="column">
            <Text dimColor>Se ejecutará:</Text>
            <Text dimColor>   $ docker exec mcp-tools-ollama ollama pull {selected}</Text>
            <Text dimColor>
              {"   · escribir "}{envVar}={selected} en .env.mem0
              {needsThinkFlag ? "  + MEM0_OLLAMA_THINK=false" : ""}
            </Text>
            <Text dimColor>   $ docker restart mcp-tools-mem0</Text>
          </Box>
          <Box marginTop={1}>
            <Text>Confirmar (y/N): </Text>
            <ConfirmInput
              onConfirm={() => {
                void runPipeline();
              }}
              onCancel={() => {
                setPhase("done");
                setTimeout(() => process.exit(0), 100);
              }}
            />
          </Box>
        </Box>
      )}

      {phase === "pulling" && (
        <Box paddingLeft={1}>
          <Spinner label={`Descargando ${selected} (puede tardar según tamaño)...`} />
        </Box>
      )}

      {phase === "restarting" && (
        <Box paddingLeft={1}>
          <Spinner label="Reiniciando mcp-tools-mem0..." />
        </Box>
      )}

      {phase === "done" && selected && kind && (
        <Box flexDirection="column">
          <Box>
            <Text backgroundColor="green" color="black" bold> OK </Text>
            <Text dimColor>  {envVar}={selected}</Text>
          </Box>
          {kind === "embed" && (
            <Box marginTop={1} flexDirection="column">
              <Text color="yellow">Aviso embeddings:</Text>
              <Text dimColor>  Si {selected} tiene dim distinta a la colección actual, se rompe.</Text>
              <Text dimColor>  Cambia MEM0_COLLECTION a un nombre nuevo, o borra la anterior:</Text>
              <Text dimColor>
                {"    $ curl -X DELETE http://127.0.0.1:6333/collections/"}
                {current.MEM0_COLLECTION ?? "<colección>"}
              </Text>
            </Box>
          )}
        </Box>
      )}

      {phase === "error" && error && (
        <Box flexDirection="column">
          <Box>
            <Text backgroundColor="red" color="black" bold> ERROR </Text>
          </Box>
          <Box marginTop={1}>
            <Text>{error}</Text>
          </Box>
        </Box>
      )}
    </Box>
  );
};

render(<App />);
