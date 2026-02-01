#!/usr/bin/env node
/**
 * run-mcps.js
 * Reads your .mcp.json and spawns each MCP server as a subprocess via stdio.
 * 
 * Usage:
 *   node run-mcps.js
 *   node run-mcps.js --only filesystem git       # only spawn these servers
 *   node run-mcps.js --exclude custom everything  # spawn all except these
 */

const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");

// ---------- CONFIG ----------
// Change this path if you moved the file somewhere else
const MCP_CONFIG_PATH = path.resolve(__dirname, ".mcp.json");

// ---------- PARSE CLI ARGS ----------
const args = process.argv.slice(2);
let filter = null; // { type: "only"|"exclude", keys: [...] }

if (args.includes("--only")) {
  const idx = args.indexOf("--only");
  filter = { type: "only", keys: args.slice(idx + 1).filter(a => !a.startsWith("--")) };
} else if (args.includes("--exclude")) {
  const idx = args.indexOf("--exclude");
  filter = { type: "exclude", keys: args.slice(idx + 1).filter(a => !a.startsWith("--")) };
}

// ---------- LOG COLORS ----------
const COLORS = {
  reset:   "\x1b[0m",
  red:     "\x1b[31m",
  green:   "\x1b[32m",
  yellow:  "\x1b[33m",
  cyan:    "\x1b[36m",
  magenta: "\x1b[35m",
};

const colorForServer = (index) => {
  const pool = [COLORS.cyan, COLORS.green, COLORS.yellow, COLORS.magenta];
  return pool[index % pool.length];
};

function log(serverName, color, message) {
  console.log(`${color}[${serverName}]${COLORS.reset} ${message}`);
}

// ---------- LOAD CONFIG ----------
if (!fs.existsSync(MCP_CONFIG_PATH)) {
  console.error(`❌ Config file not found at: ${MCP_CONFIG_PATH}`);
  console.error("   Make sure .mcp.json exists or update MCP_CONFIG_PATH in the script.");
  process.exit(1);
}

let config;
try {
  config = JSON.parse(fs.readFileSync(MCP_CONFIG_PATH, "utf-8"));
} catch (e) {
  console.error("❌ Failed to parse .mcp.json:", e.message);
  process.exit(1);
}

const servers = config.mcpServers || {};
let serverNames = Object.keys(servers);

// Apply filter if provided
if (filter) {
  if (filter.type === "only") {
    serverNames = serverNames.filter(name => filter.keys.includes(name));
  } else {
    serverNames = serverNames.filter(name => !filter.keys.includes(name));
  }
}

if (serverNames.length === 0) {
  console.error("❌ No servers match the given filter.");
  process.exit(1);
}

// ---------- SPAWN SERVERS ----------
const processes = [];

console.log(`\n🚀 Spawning ${serverNames.length} MCP server(s)...\n`);

serverNames.forEach((name, index) => {
  const serverConfig = servers[name];
  const { command, args: serverArgs = [], env: serverEnv = {} } = serverConfig;
  const color = colorForServer(index);

  log(name, color, `Command: ${command} ${serverArgs.join(" ")}`);

  const child = spawn(command, serverArgs, {
    env: { ...process.env, ...serverEnv },
    stdio: ["pipe", "pipe", "pipe"],
  });

  processes.push({ name, child, color });

  // Capture stdout
  child.stdout.on("data", (data) => {
    const lines = data.toString().trim().split("\n");
    lines.forEach(line => {
      if (line.trim()) log(name, color, line.trim());
    });
  });

  // Capture stderr
  child.stderr.on("data", (data) => {
    const lines = data.toString().trim().split("\n");
    lines.forEach(line => {
      if (line.trim()) log(name, COLORS.red, `[stderr] ${line.trim()}`);
    });
  });

  // Process exit event
  child.on("exit", (code) => {
    if (code === 0) {
      log(name, COLORS.green, "✅ Exited successfully.");
    } else {
      log(name, COLORS.red, `❌ Exited with code ${code}.`);
    }
  });

  // Spawn error event (e.g. command not found)
  child.on("error", (err) => {
    log(name, COLORS.red, `❌ Failed to start: ${err.message}`);
  });

  log(name, color, `✅ Process started (PID: ${child.pid})`);
});

// ---------- STATUS SUMMARY ----------
setTimeout(() => {
  console.log("\n─────────────────────────────────────");
  console.log("  📋 MCP Server Status");
  console.log("─────────────────────────────────────");
  processes.forEach(({ name, child, color }) => {
    const alive = !child.exitCode && child.exitCode !== 0;
    const status = alive ? `${COLORS.green}✅ Running` : `${COLORS.red}❌ Stopped`;
    console.log(`  ${color}${name.padEnd(25)}${COLORS.reset} ${status} (PID: ${child.pid})${COLORS.reset}`);
  });
  console.log("─────────────────────────────────────\n");
  console.log("  Press Ctrl+C to stop all servers.\n");
}, 2000);

// ---------- GRACEFUL SHUTDOWN ----------
function shutdown() {
  console.log("\n⏳ Shutting down servers...");
  processes.forEach(({ name, child, color }) => {
    if (!child.killed) {
      child.kill("SIGTERM");
      log(name, color, "Process terminated.");
    }
  });
  setTimeout(() => {
    console.log("✅ All servers stopped.\n");
    process.exit(0);
  }, 500);
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);