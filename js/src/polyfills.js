// QuickJS (Javy) lacks structuredClone; flint-chart calls it on plain data
// (objects, arrays, primitives, Dates). Guarded so Node keeps its native
// implementation — the Go parity test then verifies the two agree.
if (typeof globalThis.structuredClone !== "function") {
  globalThis.structuredClone = function deepClone(v) {
    if (v === null || typeof v !== "object") return v;
    if (v instanceof Date) return new Date(v.getTime());
    if (Array.isArray(v)) return v.map(deepClone);
    if (v instanceof Map) return new Map(Array.from(v, ([k, val]) => [deepClone(k), deepClone(val)]));
    if (v instanceof Set) return new Set(Array.from(v, deepClone));
    if (v instanceof RegExp) return new RegExp(v.source, v.flags);
    const out = {};
    for (const k of Object.keys(v)) out[k] = deepClone(v[k]);
    return out;
  };
}
