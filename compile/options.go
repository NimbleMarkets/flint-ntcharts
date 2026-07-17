package compile

import "github.com/NimbleMarkets/flint-ntcharts/envelope"

// Option adjusts the ChartAssemblyInput JSON before compilation.
//
// Deprecated: alias of envelope.Option; use the envelope package directly in
// new code.
type Option = envelope.Option

// WithBaseSize sets chart_spec.baseSize (terminal cells). The compiler treats
// this as a hard bound (the backend runs with maxStretch 1).
//
// Deprecated: alias of envelope.WithBaseSize; use the envelope package
// directly in new code.
var WithBaseSize = envelope.WithBaseSize

// WithCanvasSize sets chart_spec.canvasSize — the growth ceiling above baseSize.
//
// Deprecated: alias of envelope.WithCanvasSize; use the envelope package
// directly in new code.
var WithCanvasSize = envelope.WithCanvasSize
