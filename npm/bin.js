#!/usr/bin/env node
// Runs the Go binary that npm installed for this platform.
const { spawnSync } = require("node:child_process");

const pkg = `samecontact-${process.platform}-${process.arch}`;
const exe = process.platform === "win32" ? "samecontact.exe" : "samecontact";
let binary;
try {
  binary = require.resolve(`${pkg}/${exe}`);
} catch {
  console.error(`samecontact has no build for ${process.platform}-${process.arch}. Please open an issue at https://github.com/GillMacedo/samecontact/issues`);
  process.exit(2);
}
const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`samecontact: ${result.error.message}`);
  process.exit(2);
}
process.exit(result.status ?? 2);
