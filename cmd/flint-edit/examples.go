package main

// example is a built-in demo spec, loadable with ctrl+N / alt+N where N is
// its 1-based position in examples.
type example struct {
	name string // shown on the status line
	src  string // full flint ChartAssemblyInput JSON
}

// examples is ordered: index 0 loads with key 1, and seeds the editor at
// startup. Sources are compiled in (adapted from testdata/fixtures-terminal,
// not embedded — the demo must not drift when test fixtures change) and must
// compile warning-free; TestExamplesCompile enforces that.
var examples = []example{
	{name: "Bar Chart", src: `{
  "data": { "values": [
    {"month": "Jan", "revenue": 120},
    {"month": "Feb", "revenue": 180},
    {"month": "Mar", "revenue": 95},
    {"month": "Apr", "revenue": 210},
    {"month": "May", "revenue": 165}
  ]},
  "semantic_types": { "revenue": {"semanticType": "Price", "unit": "USD"} },
  "chart_spec": {
    "chartType": "Bar Chart",
    "encodings": { "x": {"field": "month"}, "y": {"field": "revenue"} }
  }
}`},
	{name: "Timeseries Line", src: `{
  "data": { "values": [
    {"date": "2026-01-01", "price": 104.2},
    {"date": "2026-02-01", "price": 108.9},
    {"date": "2026-03-01", "price": 101.4},
    {"date": "2026-04-01", "price": 115.7},
    {"date": "2026-05-01", "price": 119.3},
    {"date": "2026-06-01", "price": 112.8},
    {"date": "2026-07-01", "price": 121.5}
  ]},
  "semantic_types": { "price": {"semanticType": "Price", "unit": "USD"} },
  "chart_spec": {
    "chartType": "Line Chart",
    "chartProperties": { "includeZero_y": false },
    "encodings": { "x": {"field": "date"}, "y": {"field": "price"} }
  }
}`},
	{name: "Candlestick", src: `{
  "data": { "values": [
    {"date": "2026-01-05", "open": 100, "high": 108, "low": 97,  "close": 105},
    {"date": "2026-01-06", "open": 105, "high": 112, "low": 103, "close": 110},
    {"date": "2026-01-07", "open": 110, "high": 111, "low": 98,  "close": 99},
    {"date": "2026-01-08", "open": 99,  "high": 106, "low": 96,  "close": 104},
    {"date": "2026-01-09", "open": 104, "high": 115, "low": 102, "close": 114},
    {"date": "2026-01-12", "open": 114, "high": 118, "low": 109, "close": 111},
    {"date": "2026-01-13", "open": 111, "high": 121, "low": 110, "close": 120}
  ]},
  "chart_spec": {
    "chartType": "Candlestick Chart",
    "chartProperties": { "candleStyle": "block" },
    "encodings": {
      "x": {"field": "date"},
      "open": {"field": "open"}, "high": {"field": "high"},
      "low": {"field": "low"}, "close": {"field": "close"}
    }
  }
}`},
	{name: "Log Scale", src: `{
  "data": { "values": [
    {"month": "2026-01-01", "users": 120},
    {"month": "2026-02-01", "users": 310},
    {"month": "2026-03-01", "users": 900},
    {"month": "2026-04-01", "users": 2400},
    {"month": "2026-05-01", "users": 7100},
    {"month": "2026-06-01", "users": 19500},
    {"month": "2026-07-01", "users": 61000},
    {"month": "2026-08-01", "users": 170000}
  ]},
  "chart_spec": {
    "chartType": "Line Chart",
    "chartProperties": { "logScale_y": true },
    "encodings": { "x": {"field": "month"}, "y": {"field": "users"} }
  }
}`},
	{name: "Heatmap", src: `{
  "data": { "values": [
    {"day": "Mon", "when": "Morning",   "visits": 62},
    {"day": "Mon", "when": "Afternoon", "visits": 91},
    {"day": "Mon", "when": "Evening",   "visits": 35},
    {"day": "Tue", "when": "Morning",   "visits": 58},
    {"day": "Tue", "when": "Afternoon", "visits": 97},
    {"day": "Tue", "when": "Evening",   "visits": 41},
    {"day": "Wed", "when": "Morning",   "visits": 66},
    {"day": "Wed", "when": "Afternoon", "visits": 84},
    {"day": "Wed", "when": "Evening",   "visits": 52},
    {"day": "Thu", "when": "Morning",   "visits": 49},
    {"day": "Thu", "when": "Afternoon", "visits": 88},
    {"day": "Thu", "when": "Evening",   "visits": 60},
    {"day": "Fri", "when": "Morning",   "visits": 44},
    {"day": "Fri", "when": "Afternoon", "visits": 73},
    {"day": "Fri", "when": "Evening",   "visits": 79}
  ]},
  "chart_spec": {
    "chartType": "Heatmap",
    "encodings": { "x": {"field": "when"}, "y": {"field": "day"}, "color": {"field": "visits"} }
  }
}`},
}
