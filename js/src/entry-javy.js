import "./polyfills.js";
import { compileToVegaLite } from "./compile.js";

// Javy exposes Javy.IO for WASI stdin/stdout access.
function readStdin() {
  const chunks = [];
  let total = 0;
  while (true) {
    const buf = new Uint8Array(4096);
    const n = Javy.IO.readSync(0, buf);
    if (n < 0) throw new Error("error reading stdin");
    if (n === 0) break;
    chunks.push(buf.subarray(0, n));
    total += n;
  }
  const all = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) { all.set(c, off); off += c.length; }
  return new TextDecoder().decode(all);
}

function writeStdout(str) {
  const bytes = new TextEncoder().encode(str);
  let off = 0;
  while (off < bytes.length) {
    const n = Javy.IO.writeSync(1, bytes.subarray(off));
    off += (typeof n === "number" && n > 0) ? n : bytes.length - off;
  }
}

writeStdout(compileToVegaLite(readStdin()));
