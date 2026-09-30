// Scheme-name → concrete colors. flint's ChannelSemantics.colorScheme carries
// Vega scheme NAMES; concrete palette selection is each backend's job.
//
// The set of names must cover everything flint-chart's recommender
// (core getRecommendedColorScheme) can return; test/colormap.test.ts drives
// the recommender across the whole semantic-type registry and fails on any
// name missing here, so an upstream rename never degrades silently into the
// fallback (that happened with `blueorange` in flint-chart 0.5.1).
const CATEGORICAL: Record<string, string[]> = {
  tableau10: ["#4e79a7", "#f28e2c", "#e15759", "#76b7b2", "#59a14f",
              "#edc949", "#af7aa1", "#ff9da7", "#9c755f", "#bab0ab"],
  set1: ["#e41a1c", "#377eb8", "#4daf4a", "#984ea3", "#ff7f00",
         "#ffff33", "#a65628", "#f781bf", "#999999"],
  set2: ["#66c2a5", "#fc8d62", "#8da0cb", "#e78ac3", "#a6d854",
         "#ffd92f", "#e5c494", "#b3b3b3"],
  tableau20: ["#4e79a7", "#a0cbe8", "#f28e2c", "#ffbe7d", "#59a14f",
              "#8cd17d", "#b6992d", "#f1ce63", "#499894", "#86bcb6",
              "#e15759", "#ff9d9a", "#79706e", "#bab0ab", "#d37295",
              "#fabfd2", "#b07aa1", "#d4a6c8", "#9d7660", "#d7b5a6"],
};

// Sequential and diverging schemes share one table: both are ordered stops
// from the low end of the domain to the high end. Diverging schemes simply
// pass through a neutral midpoint. Stops are sampled from the Vega/
// ColorBrewer interpolators of the same name; a terminal gets at most a
// handful of distinguishable steps, so five to seven stops is plenty.
const SEQUENTIAL: Record<string, string[]> = {
  viridis: ["#440154", "#414487", "#2a788e", "#22a884", "#7ad151", "#fde725"],
  blues:   ["#f7fbff", "#c6dbef", "#6baed6", "#2171b5", "#08306b"],
  greens:  ["#f7fcf5", "#c7e9c0", "#74c476", "#238b45", "#00441b"],
  reds:    ["#fff5f0", "#fcbba1", "#fb6a4a", "#cb181d", "#67000d"],
  oranges: ["#fff5eb", "#fdd0a2", "#fd8d3c", "#d94801", "#7f2704"],
  purples: ["#fcfbfd", "#dadaeb", "#9e9ac8", "#6a51a3", "#3f007d"],
  yelloworangebrown: ["#ffffe5", "#fee391", "#fe9929", "#cc4c02", "#662506"],
  // Vega "goldgreen": gold at the low end through green to a deep teal.
  goldgreen: ["#f4d166", "#c8c463", "#9bb968", "#71ac6c", "#4c9c6e", "#2f8b6e", "#0e6960"],
  // diverging: cool low end, neutral midpoint, warm high end
  redblue:    ["#67001f", "#d6604d", "#f7f7f7", "#4393c3", "#053061"],
  blueorange: ["#134b73", "#4f97cc", "#b7d9ef", "#f7f7f7", "#f8d1a6", "#e08214", "#8c4109"],
};

export function isKnownScheme(scheme: string): boolean {
  return scheme in CATEGORICAL || scheme in SEQUENTIAL;
}

export function paletteForScheme(scheme: string | undefined): string[] {
  return CATEGORICAL[scheme ?? ""] ?? CATEGORICAL.tableau10;
}

export function gradientForScheme(scheme: string | undefined): string[] {
  return SEQUENTIAL[scheme ?? ""] ?? SEQUENTIAL.viridis;
}
