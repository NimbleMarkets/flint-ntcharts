// Scheme-name → concrete colors. flint's ChannelSemantics.colorScheme carries
// Vega scheme NAMES; concrete palette selection is each backend's job.
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

const SEQUENTIAL: Record<string, string[]> = {
  viridis: ["#440154", "#414487", "#2a788e", "#22a884", "#7ad151", "#fde725"],
  blues:   ["#f7fbff", "#c6dbef", "#6baed6", "#2171b5", "#08306b"],
  reds:    ["#fff5f0", "#fcbba1", "#fb6a4a", "#cb181d", "#67000d"],
  oranges: ["#fff5eb", "#fdd0a2", "#fd8d3c", "#d94801", "#7f2704"],
  purples: ["#fcfbfd", "#dadaeb", "#9e9ac8", "#6a51a3", "#3f007d"],
  yelloworangebrown: ["#ffffe5", "#fee391", "#fe9929", "#cc4c02", "#662506"],
  redblue: ["#67001f", "#d6604d", "#f7f7f7", "#4393c3", "#053061"], // diverging
};

export function paletteForScheme(scheme: string | undefined): string[] {
  return CATEGORICAL[scheme ?? ""] ?? CATEGORICAL.tableau10;
}

export function gradientForScheme(scheme: string | undefined): string[] {
  return SEQUENTIAL[scheme ?? ""] ?? SEQUENTIAL.viridis;
}
