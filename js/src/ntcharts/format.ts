import type { FormatSpec, ChartWarning } from "flint-chart/core";
import type { NtFormat } from "./types.js";

type Warn = (w: ChartWarning) => void;

// d3 pattern precision: ".2f" / ",.2f" / ".0~%" → the digit run after the dot.
function patternPrecision(pattern: string, fallback: number): number {
  const m = /\.(\d+)/.exec(pattern);
  return m ? parseInt(m[1], 10) : fallback;
}

// formatSpecToNt translates flint's FormatSpec (d3-format struct) into the
// ntcharts-spec Format directive. Suffixes have no spec equivalent and are
// dropped with a warning. Returns undefined when nothing meaningful is set,
// so callers omit the field (omitzero parity).
export function formatSpecToNt(fmt: FormatSpec | undefined, warn: Warn): NtFormat | undefined {
  if (!fmt) return undefined;
  const { pattern, prefix, suffix, abbreviate } = fmt;
  if (suffix) {
    warn({ severity: "info", code: "format-suffix-dropped",
      message: `ntcharts-spec has no format suffix; dropping ${JSON.stringify(suffix)}` });
  }
  if (prefix) {
    return { kind: "currency", currency: prefix, precision: patternPrecision(pattern ?? "", 2) };
  }
  if (pattern && pattern.includes("%")) {
    return { kind: "percent", precision: patternPrecision(pattern, 0) };
  }
  if (abbreviate) return { kind: "si", precision: 1 };
  if (pattern) {
    if (/d$/.test(pattern)) return { kind: "number", precision: 0 };
    return { kind: "number", precision: patternPrecision(pattern, 2) };
  }
  return undefined;
}

const D3_TO_GO: Array<[string, string]> = [
  ["%Y", "2006"], ["%y", "06"], ["%B", "January"], ["%b", "Jan"],
  ["%m", "01"], ["%A", "Monday"], ["%a", "Mon"], ["%d", "02"], ["%e", "_2"],
  ["%H", "15"], ["%I", "03"], ["%M", "04"], ["%S", "05"], ["%p", "PM"], ["%Z", "MST"],
];

// d3TimeToGoLayout best-effort-translates a d3 time-format string into a Go
// time layout. Unknown %-tokens pass through verbatim with one warning.
export function d3TimeToGoLayout(d3fmt: string, warn: Warn): string {
  let out = d3fmt;
  for (const [from, to] of D3_TO_GO) out = out.split(from).join(to);
  if (/%[A-Za-z]/.test(out)) {
    warn({ severity: "info", code: "time-format-partial",
      message: `d3 time format ${JSON.stringify(d3fmt)} contains untranslatable tokens; passed through` });
  }
  return out;
}
