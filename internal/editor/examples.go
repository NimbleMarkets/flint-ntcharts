package editor

// example is a built-in demo spec, loadable with ctrl+N / alt+N where N is
// its 1-based position in examples.
type example struct {
	name string // shown on the status line
	src  string // full flint ChartAssemblyInput JSON
}

// examples is ordered: index 0 loads with key 1, and seeds the editor at
// startup. Sources are compiled in (adapted from testdata/fixtures-terminal,
// not embedded — the demo must not drift when test fixtures change) and must
// compile with at most info notes; TestExamplesCompile enforces that.
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
	{name: "Histogram", src: `{
  "data": { "values": [
    {"latency_ms": 113}, {"latency_ms": 134}, {"latency_ms": 114}, {"latency_ms": 111}, {"latency_ms": 94},
    {"latency_ms": 114}, {"latency_ms": 151}, {"latency_ms": 132}, {"latency_ms": 149}, {"latency_ms": 127},
    {"latency_ms": 131}, {"latency_ms": 125}, {"latency_ms": 73}, {"latency_ms": 144}, {"latency_ms": 134},
    {"latency_ms": 134}, {"latency_ms": 73}, {"latency_ms": 71}, {"latency_ms": 95}, {"latency_ms": 107},
    {"latency_ms": 129}, {"latency_ms": 119}, {"latency_ms": 135}, {"latency_ms": 102}, {"latency_ms": 129},
    {"latency_ms": 131}, {"latency_ms": 101}, {"latency_ms": 168}, {"latency_ms": 136}, {"latency_ms": 154},
    {"latency_ms": 103}, {"latency_ms": 99}, {"latency_ms": 110}, {"latency_ms": 117}, {"latency_ms": 138},
    {"latency_ms": 127}, {"latency_ms": 107}, {"latency_ms": 93}, {"latency_ms": 105}, {"latency_ms": 154},
    {"latency_ms": 240}, {"latency_ms": 266}, {"latency_ms": 271}, {"latency_ms": 223}, {"latency_ms": 261},
    {"latency_ms": 293}, {"latency_ms": 210}, {"latency_ms": 252}, {"latency_ms": 257}, {"latency_ms": 240}
  ]},
  "chart_spec": {
    "chartType": "Histogram",
    "chartProperties": { "binCount": 12 },
    "encodings": { "x": {"field": "latency_ms"} }
  }
}`},
	{name: "Calendar Heatmap", src: `{
  "data": { "values": [
    {"day": "2026-03-02", "commits": 5}, {"day": "2026-03-03", "commits": 3}, {"day": "2026-03-04", "commits": 0}, {"day": "2026-03-05", "commits": 6}, {"day": "2026-03-06", "commits": 6}, {"day": "2026-03-07", "commits": 2}, {"day": "2026-03-08", "commits": 3},
    {"day": "2026-03-09", "commits": 5}, {"day": "2026-03-10", "commits": 4}, {"day": "2026-03-11", "commits": 0}, {"day": "2026-03-12", "commits": 5}, {"day": "2026-03-13", "commits": 2}, {"day": "2026-03-14", "commits": 0}, {"day": "2026-03-15", "commits": 0},
    {"day": "2026-03-16", "commits": 1}, {"day": "2026-03-17", "commits": 2}, {"day": "2026-03-18", "commits": 7}, {"day": "2026-03-19", "commits": 0}, {"day": "2026-03-20", "commits": 0}, {"day": "2026-03-21", "commits": 1}, {"day": "2026-03-22", "commits": 3},
    {"day": "2026-03-23", "commits": 0}, {"day": "2026-03-24", "commits": 0}, {"day": "2026-03-25", "commits": 0}, {"day": "2026-03-26", "commits": 0}, {"day": "2026-03-27", "commits": 0}, {"day": "2026-03-28", "commits": 0}, {"day": "2026-03-29", "commits": 0},
    {"day": "2026-03-30", "commits": 7}, {"day": "2026-03-31", "commits": 4}, {"day": "2026-04-01", "commits": 4}, {"day": "2026-04-02", "commits": 5}, {"day": "2026-04-03", "commits": 8}, {"day": "2026-04-04", "commits": 1}, {"day": "2026-04-05", "commits": 1},
    {"day": "2026-04-06", "commits": 5}, {"day": "2026-04-07", "commits": 0}, {"day": "2026-04-08", "commits": 7}, {"day": "2026-04-09", "commits": 6}, {"day": "2026-04-10", "commits": 5}, {"day": "2026-04-11", "commits": 0}, {"day": "2026-04-12", "commits": 0},
    {"day": "2026-04-13", "commits": 6}, {"day": "2026-04-14", "commits": 0}, {"day": "2026-04-15", "commits": 3}, {"day": "2026-04-16", "commits": 7}, {"day": "2026-04-17", "commits": 0}, {"day": "2026-04-18", "commits": 3}, {"day": "2026-04-19", "commits": 1},
    {"day": "2026-04-20", "commits": 3}, {"day": "2026-04-21", "commits": 4}, {"day": "2026-04-22", "commits": 5}, {"day": "2026-04-23", "commits": 4}, {"day": "2026-04-24", "commits": 7}, {"day": "2026-04-25", "commits": 0}, {"day": "2026-04-26", "commits": 0}
  ]},
  "chart_spec": {
    "chartType": "Calendar Heatmap",
    "encodings": { "x": {"field": "day"}, "color": {"field": "commits"} }
  }
}`},
	{name: "Pie Chart (raster)", src: `{
  "renderer": "raster",
  "data": { "values": [
    {"device": "Laptop", "units": 1190},
    {"device": "Phone",  "units": 2110},
    {"device": "Tablet", "units": 780},
    {"device": "Watch",  "units": 420}
  ]},
  "chart_spec": {
    "chartType": "Pie Chart",
    "encodings": {
      "color": {"field": "device"},
      "size": {"field": "units"}
    }
  }
}`},
	{name: "Combo Chart", src: `{
  "data": {
    "values": [
      {
        "month": "Jan",
        "revenue": 120,
        "margin": 0.2
      },
      {
        "month": "Feb",
        "revenue": 180,
        "margin": 0.25
      },
      {
        "month": "Mar",
        "revenue": 150,
        "margin": 0.22
      },
      {
        "month": "Apr",
        "revenue": 220,
        "margin": 0.3
      },
      {
        "month": "May",
        "revenue": 190,
        "margin": 0.26
      },
      {
        "month": "Jun",
        "revenue": 260,
        "margin": 0.34
      }
    ]
  },
  "semantic_types": {
    "revenue": {
      "semanticType": "Amount",
      "unit": "USD"
    },
    "margin": {
      "semanticType": "Percentage",
      "intrinsicDomain": [
        0,
        1
      ]
    }
  },
  "chart_spec": {
    "chartType": "Combo Chart",
    "title": "Revenue and margin",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "revenue"
      }
    },
    "chartProperties": {
      "lineField": "margin"
    }
  }
}`},
	{name: "Grouped Bars", src: `{
  "data": {
    "values": [
      {
        "device": "Laptop",
        "region": "USA",
        "units": 410
      },
      {
        "device": "Laptop",
        "region": "China",
        "units": 530
      },
      {
        "device": "Laptop",
        "region": "Japan",
        "units": 250
      },
      {
        "device": "Phone",
        "region": "USA",
        "units": 720
      },
      {
        "device": "Phone",
        "region": "China",
        "units": 910
      },
      {
        "device": "Phone",
        "region": "Japan",
        "units": 480
      },
      {
        "device": "Tablet",
        "region": "USA",
        "units": 190
      },
      {
        "device": "Tablet",
        "region": "China",
        "units": 260
      },
      {
        "device": "Tablet",
        "region": "Japan",
        "units": 330
      },
      {
        "device": "Watch",
        "region": "USA",
        "units": 140
      },
      {
        "device": "Watch",
        "region": "China",
        "units": 90
      },
      {
        "device": "Watch",
        "region": "Japan",
        "units": 120
      }
    ]
  },
  "semantic_types": {
    "units": {
      "semanticType": "Count"
    }
  },
  "chart_spec": {
    "chartType": "Grouped Bar Chart",
    "encodings": {
      "x": {
        "field": "device"
      },
      "y": {
        "field": "units"
      },
      "group": {
        "field": "region"
      }
    }
  }
}`},
	{name: "Pyramid", src: `{
  "data": {
    "values": [
      {
        "age": "0-14",
        "sex": "Male",
        "millions": 31
      },
      {
        "age": "0-14",
        "sex": "Female",
        "millions": 30
      },
      {
        "age": "15-29",
        "sex": "Male",
        "millions": 33
      },
      {
        "age": "15-29",
        "sex": "Female",
        "millions": 32
      },
      {
        "age": "30-44",
        "sex": "Male",
        "millions": 32
      },
      {
        "age": "30-44",
        "sex": "Female",
        "millions": 32
      },
      {
        "age": "45-59",
        "sex": "Male",
        "millions": 30
      },
      {
        "age": "45-59",
        "sex": "Female",
        "millions": 31
      },
      {
        "age": "60-74",
        "sex": "Male",
        "millions": 22
      },
      {
        "age": "60-74",
        "sex": "Female",
        "millions": 25
      },
      {
        "age": "75+",
        "sex": "Male",
        "millions": 9
      },
      {
        "age": "75+",
        "sex": "Female",
        "millions": 13
      }
    ]
  },
  "semantic_types": {
    "millions": {
      "semanticType": "Quantity"
    }
  },
  "chart_spec": {
    "chartType": "Pyramid Chart",
    "encodings": {
      "y": {
        "field": "age"
      },
      "x": {
        "field": "millions"
      },
      "color": {
        "field": "sex"
      }
    }
  }
}`},
	{name: "Density Plot", src: `{
  "data": {
    "values": [
      {
        "team": "Web",
        "latency_ms": 210
      },
      {
        "team": "Web",
        "latency_ms": 240
      },
      {
        "team": "Web",
        "latency_ms": 211
      },
      {
        "team": "Web",
        "latency_ms": 207
      },
      {
        "team": "Web",
        "latency_ms": 183
      },
      {
        "team": "Web",
        "latency_ms": 211
      },
      {
        "team": "Web",
        "latency_ms": 264
      },
      {
        "team": "Web",
        "latency_ms": 237
      },
      {
        "team": "Web",
        "latency_ms": 261
      },
      {
        "team": "Web",
        "latency_ms": 230
      },
      {
        "team": "Web",
        "latency_ms": 236
      },
      {
        "team": "Web",
        "latency_ms": 227
      },
      {
        "team": "Web",
        "latency_ms": 153
      },
      {
        "team": "Web",
        "latency_ms": 254
      },
      {
        "team": "Web",
        "latency_ms": 240
      },
      {
        "team": "Web",
        "latency_ms": 240
      },
      {
        "team": "Web",
        "latency_ms": 152
      },
      {
        "team": "Web",
        "latency_ms": 150
      },
      {
        "team": "Web",
        "latency_ms": 184
      },
      {
        "team": "Web",
        "latency_ms": 201
      },
      {
        "team": "Web",
        "latency_ms": 232
      },
      {
        "team": "Web",
        "latency_ms": 218
      },
      {
        "team": "Web",
        "latency_ms": 241
      },
      {
        "team": "Web",
        "latency_ms": 194
      },
      {
        "team": "Web",
        "latency_ms": 232
      },
      {
        "team": "Web",
        "latency_ms": 236
      },
      {
        "team": "Web",
        "latency_ms": 194
      },
      {
        "team": "Web",
        "latency_ms": 289
      },
      {
        "team": "Web",
        "latency_ms": 242
      },
      {
        "team": "Web",
        "latency_ms": 268
      },
      {
        "team": "Web",
        "latency_ms": 195
      },
      {
        "team": "Web",
        "latency_ms": 190
      },
      {
        "team": "Web",
        "latency_ms": 206
      },
      {
        "team": "Web",
        "latency_ms": 216
      },
      {
        "team": "Web",
        "latency_ms": 245
      },
      {
        "team": "Web",
        "latency_ms": 230
      },
      {
        "team": "Web",
        "latency_ms": 202
      },
      {
        "team": "Web",
        "latency_ms": 182
      },
      {
        "team": "Web",
        "latency_ms": 199
      },
      {
        "team": "Web",
        "latency_ms": 269
      },
      {
        "team": "Web",
        "latency_ms": 188
      },
      {
        "team": "Web",
        "latency_ms": 230
      },
      {
        "team": "Web",
        "latency_ms": 237
      },
      {
        "team": "Web",
        "latency_ms": 160
      },
      {
        "team": "Web",
        "latency_ms": 222
      },
      {
        "team": "Web",
        "latency_ms": 272
      },
      {
        "team": "Web",
        "latency_ms": 139
      },
      {
        "team": "Web",
        "latency_ms": 207
      },
      {
        "team": "Web",
        "latency_ms": 216
      },
      {
        "team": "Web",
        "latency_ms": 187
      },
      {
        "team": "Web",
        "latency_ms": 240
      },
      {
        "team": "Web",
        "latency_ms": 218
      },
      {
        "team": "Web",
        "latency_ms": 161
      },
      {
        "team": "Web",
        "latency_ms": 253
      },
      {
        "team": "Web",
        "latency_ms": 247
      },
      {
        "team": "Web",
        "latency_ms": 258
      },
      {
        "team": "Web",
        "latency_ms": 278
      },
      {
        "team": "Web",
        "latency_ms": 234
      },
      {
        "team": "Web",
        "latency_ms": 225
      },
      {
        "team": "Web",
        "latency_ms": 168
      },
      {
        "team": "Mobile",
        "latency_ms": 347
      },
      {
        "team": "Mobile",
        "latency_ms": 273
      },
      {
        "team": "Mobile",
        "latency_ms": 283
      },
      {
        "team": "Mobile",
        "latency_ms": 234
      },
      {
        "team": "Mobile",
        "latency_ms": 252
      },
      {
        "team": "Mobile",
        "latency_ms": 278
      },
      {
        "team": "Mobile",
        "latency_ms": 387
      },
      {
        "team": "Mobile",
        "latency_ms": 188
      },
      {
        "team": "Mobile",
        "latency_ms": 223
      },
      {
        "team": "Mobile",
        "latency_ms": 324
      },
      {
        "team": "Mobile",
        "latency_ms": 397
      },
      {
        "team": "Mobile",
        "latency_ms": 345
      },
      {
        "team": "Mobile",
        "latency_ms": 196
      },
      {
        "team": "Mobile",
        "latency_ms": 159
      },
      {
        "team": "Mobile",
        "latency_ms": 331
      },
      {
        "team": "Mobile",
        "latency_ms": 266
      },
      {
        "team": "Mobile",
        "latency_ms": 243
      },
      {
        "team": "Mobile",
        "latency_ms": 369
      },
      {
        "team": "Mobile",
        "latency_ms": 376
      },
      {
        "team": "Mobile",
        "latency_ms": 319
      },
      {
        "team": "Mobile",
        "latency_ms": 325
      },
      {
        "team": "Mobile",
        "latency_ms": 336
      },
      {
        "team": "Mobile",
        "latency_ms": 406
      },
      {
        "team": "Mobile",
        "latency_ms": 347
      },
      {
        "team": "Mobile",
        "latency_ms": 341
      },
      {
        "team": "Mobile",
        "latency_ms": 343
      },
      {
        "team": "Mobile",
        "latency_ms": 216
      },
      {
        "team": "Mobile",
        "latency_ms": 387
      },
      {
        "team": "Mobile",
        "latency_ms": 367
      },
      {
        "team": "Mobile",
        "latency_ms": 342
      },
      {
        "team": "Mobile",
        "latency_ms": 192
      },
      {
        "team": "Mobile",
        "latency_ms": 272
      },
      {
        "team": "Mobile",
        "latency_ms": 361
      },
      {
        "team": "Mobile",
        "latency_ms": 201
      },
      {
        "team": "Mobile",
        "latency_ms": 299
      },
      {
        "team": "Mobile",
        "latency_ms": 371
      },
      {
        "team": "Mobile",
        "latency_ms": 231
      },
      {
        "team": "Mobile",
        "latency_ms": 407
      },
      {
        "team": "Mobile",
        "latency_ms": 343
      },
      {
        "team": "Mobile",
        "latency_ms": 301
      },
      {
        "team": "Mobile",
        "latency_ms": 329
      },
      {
        "team": "Mobile",
        "latency_ms": 349
      },
      {
        "team": "Mobile",
        "latency_ms": 317
      },
      {
        "team": "Mobile",
        "latency_ms": 379
      },
      {
        "team": "Mobile",
        "latency_ms": 270
      },
      {
        "team": "Mobile",
        "latency_ms": 285
      },
      {
        "team": "Mobile",
        "latency_ms": 373
      },
      {
        "team": "Mobile",
        "latency_ms": 312
      },
      {
        "team": "Mobile",
        "latency_ms": 257
      },
      {
        "team": "Mobile",
        "latency_ms": 367
      },
      {
        "team": "Mobile",
        "latency_ms": 398
      },
      {
        "team": "Mobile",
        "latency_ms": 283
      },
      {
        "team": "Mobile",
        "latency_ms": 227
      },
      {
        "team": "Mobile",
        "latency_ms": 302
      },
      {
        "team": "Mobile",
        "latency_ms": 301
      },
      {
        "team": "Mobile",
        "latency_ms": 292
      },
      {
        "team": "Mobile",
        "latency_ms": 394
      },
      {
        "team": "Mobile",
        "latency_ms": 248
      },
      {
        "team": "Mobile",
        "latency_ms": 386
      },
      {
        "team": "Mobile",
        "latency_ms": 234
      }
    ]
  },
  "semantic_types": {
    "latency_ms": {
      "semanticType": "Duration",
      "unit": "ms"
    }
  },
  "chart_spec": {
    "chartType": "Density Plot",
    "encodings": {
      "x": {
        "field": "latency_ms"
      },
      "color": {
        "field": "team"
      }
    }
  }
}`},
	{name: "Range Area", src: `{
  "data": {
    "values": [
      {
        "month": "Jan",
        "low": 2,
        "high": 8
      },
      {
        "month": "Feb",
        "low": 3,
        "high": 10
      },
      {
        "month": "Mar",
        "low": 5,
        "high": 13
      },
      {
        "month": "Apr",
        "low": 7,
        "high": 16
      },
      {
        "month": "May",
        "low": 10,
        "high": 20
      },
      {
        "month": "Jun",
        "low": 13,
        "high": 23
      },
      {
        "month": "Jul",
        "low": 15,
        "high": 26
      },
      {
        "month": "Aug",
        "low": 15,
        "high": 26
      },
      {
        "month": "Sep",
        "low": 12,
        "high": 22
      },
      {
        "month": "Oct",
        "low": 9,
        "high": 16
      },
      {
        "month": "Nov",
        "low": 5,
        "high": 11
      },
      {
        "month": "Dec",
        "low": 3,
        "high": 8
      }
    ]
  },
  "semantic_types": {
    "month": {
      "semanticType": "Month"
    },
    "low": {
      "semanticType": "Temperature",
      "unit": "°C"
    },
    "high": {
      "semanticType": "Temperature",
      "unit": "°C"
    }
  },
  "chart_spec": {
    "chartType": "Range Area Chart",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "low"
      },
      "y2": {
        "field": "high"
      }
    }
  }
}`},
	{name: "Regression", src: `{
  "data": {
    "values": [
      {
        "hours": 1,
        "score": 36
      },
      {
        "hours": 2,
        "score": 50
      },
      {
        "hours": 2,
        "score": 53
      },
      {
        "hours": 3,
        "score": 57
      },
      {
        "hours": 3,
        "score": 54
      },
      {
        "hours": 4,
        "score": 58
      },
      {
        "hours": 4,
        "score": 58
      },
      {
        "hours": 5,
        "score": 66
      },
      {
        "hours": 5,
        "score": 61
      },
      {
        "hours": 6,
        "score": 70
      },
      {
        "hours": 6,
        "score": 71
      },
      {
        "hours": 7,
        "score": 74
      },
      {
        "hours": 7,
        "score": 78
      },
      {
        "hours": 8,
        "score": 82
      },
      {
        "hours": 8,
        "score": 91
      },
      {
        "hours": 9,
        "score": 86
      },
      {
        "hours": 9,
        "score": 82
      },
      {
        "hours": 10,
        "score": 88
      }
    ]
  },
  "semantic_types": {
    "hours": {
      "semanticType": "Duration",
      "unit": "hours"
    },
    "score": {
      "semanticType": "Score"
    }
  },
  "chart_spec": {
    "chartType": "Regression",
    "encodings": {
      "x": {
        "field": "hours"
      },
      "y": {
        "field": "score"
      }
    }
  }
}`},
	{name: "Stacked Bars", src: `{
  "data": { "values": [
    {"month": "2024-01-01", "product": "Cloud", "revenue": 409},
    {"month": "2024-01-01", "product": "Licences", "revenue": 289},
    {"month": "2024-01-01", "product": "Services", "revenue": 189},
    {"month": "2024-02-01", "product": "Cloud", "revenue": 408},
    {"month": "2024-02-01", "product": "Licences", "revenue": 309},
    {"month": "2024-02-01", "product": "Services", "revenue": 176},
    {"month": "2024-03-01", "product": "Cloud", "revenue": 421},
    {"month": "2024-03-01", "product": "Licences", "revenue": 304},
    {"month": "2024-03-01", "product": "Services", "revenue": 160},
    {"month": "2024-04-01", "product": "Cloud", "revenue": 458},
    {"month": "2024-04-01", "product": "Licences", "revenue": 275},
    {"month": "2024-04-01", "product": "Services", "revenue": 167},
    {"month": "2024-05-01", "product": "Cloud", "revenue": 471},
    {"month": "2024-05-01", "product": "Licences", "revenue": 318},
    {"month": "2024-05-01", "product": "Services", "revenue": 173},
    {"month": "2024-06-01", "product": "Cloud", "revenue": 473},
    {"month": "2024-06-01", "product": "Licences", "revenue": 303},
    {"month": "2024-06-01", "product": "Services", "revenue": 227},
    {"month": "2024-07-01", "product": "Cloud", "revenue": 509},
    {"month": "2024-07-01", "product": "Licences", "revenue": 286},
    {"month": "2024-07-01", "product": "Services", "revenue": 233},
    {"month": "2024-08-01", "product": "Cloud", "revenue": 491},
    {"month": "2024-08-01", "product": "Licences", "revenue": 311},
    {"month": "2024-08-01", "product": "Services", "revenue": 195},
    {"month": "2024-09-01", "product": "Cloud", "revenue": 511},
    {"month": "2024-09-01", "product": "Licences", "revenue": 263},
    {"month": "2024-09-01", "product": "Services", "revenue": 201},
    {"month": "2024-10-01", "product": "Cloud", "revenue": 565},
    {"month": "2024-10-01", "product": "Licences", "revenue": 264},
    {"month": "2024-10-01", "product": "Services", "revenue": 221},
    {"month": "2024-11-01", "product": "Cloud", "revenue": 568},
    {"month": "2024-11-01", "product": "Licences", "revenue": 272},
    {"month": "2024-11-01", "product": "Services", "revenue": 223},
    {"month": "2024-12-01", "product": "Cloud", "revenue": 548},
    {"month": "2024-12-01", "product": "Licences", "revenue": 251},
    {"month": "2024-12-01", "product": "Services", "revenue": 206},
    {"month": "2025-01-01", "product": "Cloud", "revenue": 599},
    {"month": "2025-01-01", "product": "Licences", "revenue": 270},
    {"month": "2025-01-01", "product": "Services", "revenue": 217},
    {"month": "2025-02-01", "product": "Cloud", "revenue": 607},
    {"month": "2025-02-01", "product": "Licences", "revenue": 268},
    {"month": "2025-02-01", "product": "Services", "revenue": 220},
    {"month": "2025-03-01", "product": "Cloud", "revenue": 634},
    {"month": "2025-03-01", "product": "Licences", "revenue": 280},
    {"month": "2025-03-01", "product": "Services", "revenue": 221},
    {"month": "2025-04-01", "product": "Cloud", "revenue": 634},
    {"month": "2025-04-01", "product": "Licences", "revenue": 267},
    {"month": "2025-04-01", "product": "Services", "revenue": 263},
    {"month": "2025-05-01", "product": "Cloud", "revenue": 658},
    {"month": "2025-05-01", "product": "Licences", "revenue": 249},
    {"month": "2025-05-01", "product": "Services", "revenue": 273},
    {"month": "2025-06-01", "product": "Cloud", "revenue": 635},
    {"month": "2025-06-01", "product": "Licences", "revenue": 254},
    {"month": "2025-06-01", "product": "Services", "revenue": 263}
  ]},
  "semantic_types": {
    "month": "Date",
    "product": "Category",
    "revenue": {
      "semanticType": "Amount",
      "unit": "USD"
    }
  },
  "chart_spec": {
    "chartType": "Stacked Bar Chart",
    "title": "Revenue by product",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "revenue"
      },
      "color": {
        "field": "product"
      }
    }
  }
}`},
	{name: "Long Names", src: `{
  "data": { "values": [
    {"editor": "Visual Studio Code", "share": 73.6},
    {"editor": "Visual Studio", "share": 28.4},
    {"editor": "IntelliJ IDEA", "share": 26.8},
    {"editor": "Notepad++", "share": 24.3},
    {"editor": "Vim", "share": 21.6},
    {"editor": "Android Studio", "share": 16.1},
    {"editor": "Jupyter Notebook / JupyterLab", "share": 12.8},
    {"editor": "Neovim", "share": 11.9}
  ]},
  "semantic_types": {
    "editor": "Category",
    "share": {
      "semanticType": "Percentage",
      "unit": "%"
    }
  },
  "chart_spec": {
    "chartType": "Bar Chart",
    "title": "Most used code editors (% of developers)",
    "encodings": {
      "x": {
        "field": "editor",
        "sortBy": "y",
        "sortOrder": "descending"
      },
      "y": {
        "field": "share"
      }
    }
  }
}`},
	{name: "Net Cash Flow", src: `{
  "data": { "values": [
    {"month": "Jan", "net": 42000},
    {"month": "Feb", "net": 18000},
    {"month": "Mar", "net": -25000},
    {"month": "Apr", "net": -61000},
    {"month": "May", "net": -12000},
    {"month": "Jun", "net": 30000},
    {"month": "Jul", "net": 55000},
    {"month": "Aug", "net": 71000},
    {"month": "Sep", "net": 12000},
    {"month": "Oct", "net": -8000},
    {"month": "Nov", "net": -34000},
    {"month": "Dec", "net": 47000}
  ]},
  "semantic_types": {
    "month": "Month",
    "net": {
      "semanticType": "Profit",
      "unit": "USD"
    }
  },
  "chart_spec": {
    "chartType": "Bar Chart",
    "title": "Net cash flow, 2025",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "net"
      }
    }
  }
}`},
	{name: "Hourly Traffic", src: `{
  "data": { "values": [
    {"hour": "00:00", "requests": 1200},
    {"hour": "01:00", "requests": 800},
    {"hour": "02:00", "requests": 600},
    {"hour": "03:00", "requests": 500},
    {"hour": "04:00", "requests": 600},
    {"hour": "05:00", "requests": 1100},
    {"hour": "06:00", "requests": 2900},
    {"hour": "07:00", "requests": 6400},
    {"hour": "08:00", "requests": 9800},
    {"hour": "09:00", "requests": 11200},
    {"hour": "10:00", "requests": 10400},
    {"hour": "11:00", "requests": 9600},
    {"hour": "12:00", "requests": 10100},
    {"hour": "13:00", "requests": 10800},
    {"hour": "14:00", "requests": 9900},
    {"hour": "15:00", "requests": 9400},
    {"hour": "16:00", "requests": 10200},
    {"hour": "17:00", "requests": 12100},
    {"hour": "18:00", "requests": 13300},
    {"hour": "19:00", "requests": 11800},
    {"hour": "20:00", "requests": 9200},
    {"hour": "21:00", "requests": 6700},
    {"hour": "22:00", "requests": 4100},
    {"hour": "23:00", "requests": 2200}
  ]},
  "semantic_types": {
    "hour": "Time",
    "requests": "Count"
  },
  "chart_spec": {
    "chartType": "Bar Chart",
    "title": "API requests by hour",
    "encodings": {
      "x": {
        "field": "hour"
      },
      "y": {
        "field": "requests"
      }
    }
  }
}`},
	{name: "Scatter Groups", src: `{
  "data": { "values": [
    {"horsepower": 169, "mpg": 18.9, "origin": "USA"},
    {"horsepower": 190, "mpg": 15.7, "origin": "USA"},
    {"horsepower": 153, "mpg": 14.5, "origin": "USA"},
    {"horsepower": 167, "mpg": 15.1, "origin": "USA"},
    {"horsepower": 137, "mpg": 15.9, "origin": "USA"},
    {"horsepower": 123, "mpg": 18.8, "origin": "USA"},
    {"horsepower": 186, "mpg": 10.0, "origin": "USA"},
    {"horsepower": 109, "mpg": 21.9, "origin": "USA"},
    {"horsepower": 190, "mpg": 16.2, "origin": "USA"},
    {"horsepower": 97, "mpg": 16.0, "origin": "USA"},
    {"horsepower": 160, "mpg": 15.4, "origin": "USA"},
    {"horsepower": 119, "mpg": 23.0, "origin": "USA"},
    {"horsepower": 181, "mpg": 15.9, "origin": "USA"},
    {"horsepower": 157, "mpg": 18.5, "origin": "USA"},
    {"horsepower": 140, "mpg": 25.0, "origin": "Europe"},
    {"horsepower": 110, "mpg": 27.2, "origin": "Europe"},
    {"horsepower": 51, "mpg": 33.7, "origin": "Europe"},
    {"horsepower": 122, "mpg": 26.2, "origin": "Europe"},
    {"horsepower": 40, "mpg": 29.8, "origin": "Europe"},
    {"horsepower": 119, "mpg": 20.6, "origin": "Europe"},
    {"horsepower": 90, "mpg": 30.0, "origin": "Europe"},
    {"horsepower": 58, "mpg": 34.0, "origin": "Europe"},
    {"horsepower": 110, "mpg": 25.4, "origin": "Europe"},
    {"horsepower": 104, "mpg": 27.9, "origin": "Europe"},
    {"horsepower": 98, "mpg": 29.6, "origin": "Europe"},
    {"horsepower": 76, "mpg": 27.4, "origin": "Europe"},
    {"horsepower": 124, "mpg": 24.7, "origin": "Europe"},
    {"horsepower": 70, "mpg": 31.3, "origin": "Europe"},
    {"horsepower": 126, "mpg": 26.6, "origin": "Japan"},
    {"horsepower": 46, "mpg": 33.8, "origin": "Japan"},
    {"horsepower": 81, "mpg": 30.6, "origin": "Japan"},
    {"horsepower": 124, "mpg": 25.3, "origin": "Japan"},
    {"horsepower": 120, "mpg": 25.0, "origin": "Japan"},
    {"horsepower": 63, "mpg": 34.3, "origin": "Japan"},
    {"horsepower": 117, "mpg": 30.6, "origin": "Japan"},
    {"horsepower": 95, "mpg": 30.6, "origin": "Japan"},
    {"horsepower": 89, "mpg": 32.1, "origin": "Japan"},
    {"horsepower": 80, "mpg": 32.1, "origin": "Japan"},
    {"horsepower": 101, "mpg": 29.7, "origin": "Japan"},
    {"horsepower": 106, "mpg": 30.7, "origin": "Japan"},
    {"horsepower": 141, "mpg": 27.3, "origin": "Japan"},
    {"horsepower": 73, "mpg": 31.0, "origin": "Japan"}
  ]},
  "semantic_types": {
    "horsepower": "Quantity",
    "mpg": "Quantity",
    "origin": "Category"
  },
  "chart_spec": {
    "chartType": "Scatter Plot",
    "title": "Fuel economy vs power",
    "encodings": {
      "x": {
        "field": "horsepower"
      },
      "y": {
        "field": "mpg"
      },
      "color": {
        "field": "origin"
      }
    }
  }
}`},
	{name: "Theme (Economist)", src: `{
  "data": { "values": [
    {"month": "2025-01-01", "region": "North", "orders": 92},
    {"month": "2025-02-01", "region": "North", "orders": 93},
    {"month": "2025-03-01", "region": "North", "orders": 94},
    {"month": "2025-04-01", "region": "North", "orders": 93},
    {"month": "2025-05-01", "region": "North", "orders": 98},
    {"month": "2025-06-01", "region": "North", "orders": 104},
    {"month": "2025-07-01", "region": "North", "orders": 106},
    {"month": "2025-08-01", "region": "North", "orders": 108},
    {"month": "2025-09-01", "region": "North", "orders": 107},
    {"month": "2025-10-01", "region": "North", "orders": 106},
    {"month": "2025-11-01", "region": "North", "orders": 106},
    {"month": "2025-12-01", "region": "North", "orders": 107},
    {"month": "2025-01-01", "region": "South", "orders": 84},
    {"month": "2025-02-01", "region": "South", "orders": 82},
    {"month": "2025-03-01", "region": "South", "orders": 80},
    {"month": "2025-04-01", "region": "South", "orders": 85},
    {"month": "2025-05-01", "region": "South", "orders": 86},
    {"month": "2025-06-01", "region": "South", "orders": 84},
    {"month": "2025-07-01", "region": "South", "orders": 86},
    {"month": "2025-08-01", "region": "South", "orders": 83},
    {"month": "2025-09-01", "region": "South", "orders": 85},
    {"month": "2025-10-01", "region": "South", "orders": 90},
    {"month": "2025-11-01", "region": "South", "orders": 94},
    {"month": "2025-12-01", "region": "South", "orders": 97},
    {"month": "2025-01-01", "region": "East", "orders": 71},
    {"month": "2025-02-01", "region": "East", "orders": 73},
    {"month": "2025-03-01", "region": "East", "orders": 73},
    {"month": "2025-04-01", "region": "East", "orders": 79},
    {"month": "2025-05-01", "region": "East", "orders": 82},
    {"month": "2025-06-01", "region": "East", "orders": 87},
    {"month": "2025-07-01", "region": "East", "orders": 89},
    {"month": "2025-08-01", "region": "East", "orders": 89},
    {"month": "2025-09-01", "region": "East", "orders": 95},
    {"month": "2025-10-01", "region": "East", "orders": 102},
    {"month": "2025-11-01", "region": "East", "orders": 108},
    {"month": "2025-12-01", "region": "East", "orders": 113},
    {"month": "2025-01-01", "region": "West", "orders": 104},
    {"month": "2025-02-01", "region": "West", "orders": 107},
    {"month": "2025-03-01", "region": "West", "orders": 105},
    {"month": "2025-04-01", "region": "West", "orders": 106},
    {"month": "2025-05-01", "region": "West", "orders": 105},
    {"month": "2025-06-01", "region": "West", "orders": 102},
    {"month": "2025-07-01", "region": "West", "orders": 99},
    {"month": "2025-08-01", "region": "West", "orders": 98},
    {"month": "2025-09-01", "region": "West", "orders": 96},
    {"month": "2025-10-01", "region": "West", "orders": 99},
    {"month": "2025-11-01", "region": "West", "orders": 103},
    {"month": "2025-12-01", "region": "West", "orders": 103},
    {"month": "2025-01-01", "region": "Central", "orders": 70},
    {"month": "2025-02-01", "region": "Central", "orders": 77},
    {"month": "2025-03-01", "region": "Central", "orders": 83},
    {"month": "2025-04-01", "region": "Central", "orders": 84},
    {"month": "2025-05-01", "region": "Central", "orders": 85},
    {"month": "2025-06-01", "region": "Central", "orders": 85},
    {"month": "2025-07-01", "region": "Central", "orders": 85},
    {"month": "2025-08-01", "region": "Central", "orders": 86},
    {"month": "2025-09-01", "region": "Central", "orders": 89},
    {"month": "2025-10-01", "region": "Central", "orders": 95},
    {"month": "2025-11-01", "region": "Central", "orders": 100},
    {"month": "2025-12-01", "region": "Central", "orders": 103}
  ]},
  "semantic_types": {
    "month": "Date",
    "region": "Category",
    "orders": "Count"
  },
  "theme_spec": "economist",
  "chart_spec": {
    "chartType": "Line Chart",
    "title": "Orders by region",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "orders"
      },
      "color": {
        "field": "region"
      }
    }
  }
}`},
	{name: "Streamgraph", src: `{
  "data": { "values": [
    {"week": "2025-01-06", "genre": "Pop", "hours": 122},
    {"week": "2025-01-06", "genre": "Hip-hop", "hours": 118},
    {"week": "2025-01-06", "genre": "Rock", "hours": 68},
    {"week": "2025-01-06", "genre": "Electronic", "hours": 12},
    {"week": "2025-01-06", "genre": "Jazz", "hours": 31},
    {"week": "2025-01-13", "genre": "Pop", "hours": 130},
    {"week": "2025-01-13", "genre": "Hip-hop", "hours": 117},
    {"week": "2025-01-13", "genre": "Rock", "hours": 68},
    {"week": "2025-01-13", "genre": "Electronic", "hours": 7},
    {"week": "2025-01-13", "genre": "Jazz", "hours": 29},
    {"week": "2025-01-20", "genre": "Pop", "hours": 133},
    {"week": "2025-01-20", "genre": "Hip-hop", "hours": 116},
    {"week": "2025-01-20", "genre": "Rock", "hours": 69},
    {"week": "2025-01-20", "genre": "Electronic", "hours": 10},
    {"week": "2025-01-20", "genre": "Jazz", "hours": 24},
    {"week": "2025-01-27", "genre": "Pop", "hours": 145},
    {"week": "2025-01-27", "genre": "Hip-hop", "hours": 112},
    {"week": "2025-01-27", "genre": "Rock", "hours": 58},
    {"week": "2025-01-27", "genre": "Electronic", "hours": 11},
    {"week": "2025-01-27", "genre": "Jazz", "hours": 20},
    {"week": "2025-02-03", "genre": "Pop", "hours": 149},
    {"week": "2025-02-03", "genre": "Hip-hop", "hours": 108},
    {"week": "2025-02-03", "genre": "Rock", "hours": 55},
    {"week": "2025-02-03", "genre": "Electronic", "hours": 22},
    {"week": "2025-02-03", "genre": "Jazz", "hours": 26},
    {"week": "2025-02-10", "genre": "Pop", "hours": 150},
    {"week": "2025-02-10", "genre": "Hip-hop", "hours": 98},
    {"week": "2025-02-10", "genre": "Rock", "hours": 57},
    {"week": "2025-02-10", "genre": "Electronic", "hours": 21},
    {"week": "2025-02-10", "genre": "Jazz", "hours": 14},
    {"week": "2025-02-17", "genre": "Pop", "hours": 155},
    {"week": "2025-02-17", "genre": "Hip-hop", "hours": 95},
    {"week": "2025-02-17", "genre": "Rock", "hours": 56},
    {"week": "2025-02-17", "genre": "Electronic", "hours": 36},
    {"week": "2025-02-17", "genre": "Jazz", "hours": 17},
    {"week": "2025-02-24", "genre": "Pop", "hours": 153},
    {"week": "2025-02-24", "genre": "Hip-hop", "hours": 91},
    {"week": "2025-02-24", "genre": "Rock", "hours": 52},
    {"week": "2025-02-24", "genre": "Electronic", "hours": 37},
    {"week": "2025-02-24", "genre": "Jazz", "hours": 13},
    {"week": "2025-03-03", "genre": "Pop", "hours": 145},
    {"week": "2025-03-03", "genre": "Hip-hop", "hours": 82},
    {"week": "2025-03-03", "genre": "Rock", "hours": 53},
    {"week": "2025-03-03", "genre": "Electronic", "hours": 46},
    {"week": "2025-03-03", "genre": "Jazz", "hours": 10},
    {"week": "2025-03-10", "genre": "Pop", "hours": 147},
    {"week": "2025-03-10", "genre": "Hip-hop", "hours": 74},
    {"week": "2025-03-10", "genre": "Rock", "hours": 57},
    {"week": "2025-03-10", "genre": "Electronic", "hours": 54},
    {"week": "2025-03-10", "genre": "Jazz", "hours": 17},
    {"week": "2025-03-17", "genre": "Pop", "hours": 137},
    {"week": "2025-03-17", "genre": "Hip-hop", "hours": 75},
    {"week": "2025-03-17", "genre": "Rock", "hours": 59},
    {"week": "2025-03-17", "genre": "Electronic", "hours": 60},
    {"week": "2025-03-17", "genre": "Jazz", "hours": 12},
    {"week": "2025-03-24", "genre": "Pop", "hours": 127},
    {"week": "2025-03-24", "genre": "Hip-hop", "hours": 67},
    {"week": "2025-03-24", "genre": "Rock", "hours": 59},
    {"week": "2025-03-24", "genre": "Electronic", "hours": 60},
    {"week": "2025-03-24", "genre": "Jazz", "hours": 15},
    {"week": "2025-03-31", "genre": "Pop", "hours": 121},
    {"week": "2025-03-31", "genre": "Hip-hop", "hours": 65},
    {"week": "2025-03-31", "genre": "Rock", "hours": 68},
    {"week": "2025-03-31", "genre": "Electronic", "hours": 69},
    {"week": "2025-03-31", "genre": "Jazz", "hours": 11},
    {"week": "2025-04-07", "genre": "Pop", "hours": 117},
    {"week": "2025-04-07", "genre": "Hip-hop", "hours": 66},
    {"week": "2025-04-07", "genre": "Rock", "hours": 72},
    {"week": "2025-04-07", "genre": "Electronic", "hours": 66},
    {"week": "2025-04-07", "genre": "Jazz", "hours": 14},
    {"week": "2025-04-14", "genre": "Pop", "hours": 107},
    {"week": "2025-04-14", "genre": "Hip-hop", "hours": 64},
    {"week": "2025-04-14", "genre": "Rock", "hours": 76},
    {"week": "2025-04-14", "genre": "Electronic", "hours": 70},
    {"week": "2025-04-14", "genre": "Jazz", "hours": 15},
    {"week": "2025-04-21", "genre": "Pop", "hours": 105},
    {"week": "2025-04-21", "genre": "Hip-hop", "hours": 73},
    {"week": "2025-04-21", "genre": "Rock", "hours": 76},
    {"week": "2025-04-21", "genre": "Electronic", "hours": 69},
    {"week": "2025-04-21", "genre": "Jazz", "hours": 16},
    {"week": "2025-04-28", "genre": "Pop", "hours": 97},
    {"week": "2025-04-28", "genre": "Hip-hop", "hours": 74},
    {"week": "2025-04-28", "genre": "Rock", "hours": 79},
    {"week": "2025-04-28", "genre": "Electronic", "hours": 64},
    {"week": "2025-04-28", "genre": "Jazz", "hours": 18},
    {"week": "2025-05-05", "genre": "Pop", "hours": 98},
    {"week": "2025-05-05", "genre": "Hip-hop", "hours": 79},
    {"week": "2025-05-05", "genre": "Rock", "hours": 86},
    {"week": "2025-05-05", "genre": "Electronic", "hours": 63},
    {"week": "2025-05-05", "genre": "Jazz", "hours": 17},
    {"week": "2025-05-12", "genre": "Pop", "hours": 91},
    {"week": "2025-05-12", "genre": "Hip-hop", "hours": 87},
    {"week": "2025-05-12", "genre": "Rock", "hours": 87},
    {"week": "2025-05-12", "genre": "Electronic", "hours": 49},
    {"week": "2025-05-12", "genre": "Jazz", "hours": 18},
    {"week": "2025-05-19", "genre": "Pop", "hours": 89},
    {"week": "2025-05-19", "genre": "Hip-hop", "hours": 85},
    {"week": "2025-05-19", "genre": "Rock", "hours": 82},
    {"week": "2025-05-19", "genre": "Electronic", "hours": 41},
    {"week": "2025-05-19", "genre": "Jazz", "hours": 25},
    {"week": "2025-05-26", "genre": "Pop", "hours": 94},
    {"week": "2025-05-26", "genre": "Hip-hop", "hours": 99},
    {"week": "2025-05-26", "genre": "Rock", "hours": 81},
    {"week": "2025-05-26", "genre": "Electronic", "hours": 40},
    {"week": "2025-05-26", "genre": "Jazz", "hours": 27},
    {"week": "2025-06-02", "genre": "Pop", "hours": 91},
    {"week": "2025-06-02", "genre": "Hip-hop", "hours": 105},
    {"week": "2025-06-02", "genre": "Rock", "hours": 89},
    {"week": "2025-06-02", "genre": "Electronic", "hours": 28},
    {"week": "2025-06-02", "genre": "Jazz", "hours": 31},
    {"week": "2025-06-09", "genre": "Pop", "hours": 98},
    {"week": "2025-06-09", "genre": "Hip-hop", "hours": 106},
    {"week": "2025-06-09", "genre": "Rock", "hours": 87},
    {"week": "2025-06-09", "genre": "Electronic", "hours": 27},
    {"week": "2025-06-09", "genre": "Jazz", "hours": 24},
    {"week": "2025-06-16", "genre": "Pop", "hours": 104},
    {"week": "2025-06-16", "genre": "Hip-hop", "hours": 111},
    {"week": "2025-06-16", "genre": "Rock", "hours": 78},
    {"week": "2025-06-16", "genre": "Electronic", "hours": 15},
    {"week": "2025-06-16", "genre": "Jazz", "hours": 26},
    {"week": "2025-06-23", "genre": "Pop", "hours": 114},
    {"week": "2025-06-23", "genre": "Hip-hop", "hours": 109},
    {"week": "2025-06-23", "genre": "Rock", "hours": 77},
    {"week": "2025-06-23", "genre": "Electronic", "hours": 13},
    {"week": "2025-06-23", "genre": "Jazz", "hours": 23},
    {"week": "2025-06-30", "genre": "Pop", "hours": 117},
    {"week": "2025-06-30", "genre": "Hip-hop", "hours": 116},
    {"week": "2025-06-30", "genre": "Rock", "hours": 73},
    {"week": "2025-06-30", "genre": "Electronic", "hours": 7},
    {"week": "2025-06-30", "genre": "Jazz", "hours": 32}
  ]},
  "semantic_types": {
    "week": "Date",
    "genre": "Category",
    "hours": "Quantity"
  },
  "chart_spec": {
    "chartType": "Streamgraph",
    "title": "Listening hours by genre",
    "encodings": {
      "x": {
        "field": "week"
      },
      "y": {
        "field": "hours"
      },
      "color": {
        "field": "genre"
      }
    }
  }
}`},
	{name: "Slope Chart", src: `{
  "data": { "values": [
    {"year": "2019", "industry": "Tech", "remote": 22},
    {"year": "2024", "industry": "Tech", "remote": 64},
    {"year": "2019", "industry": "Finance", "remote": 12},
    {"year": "2024", "industry": "Finance", "remote": 38},
    {"year": "2019", "industry": "Education", "remote": 18},
    {"year": "2024", "industry": "Education", "remote": 21},
    {"year": "2019", "industry": "Healthcare", "remote": 9},
    {"year": "2024", "industry": "Healthcare", "remote": 14},
    {"year": "2019", "industry": "Retail", "remote": 14},
    {"year": "2024", "industry": "Retail", "remote": 7}
  ]},
  "semantic_types": {
    "year": "Category",
    "industry": "Category",
    "remote": {
      "semanticType": "Percentage",
      "unit": "%"
    }
  },
  "chart_spec": {
    "chartType": "Slope Chart",
    "title": "Share working remotely, 2019 vs 2024 (%)",
    "encodings": {
      "x": {
        "field": "year"
      },
      "y": {
        "field": "remote"
      },
      "color": {
        "field": "industry"
      }
    }
  }
}`},
	{name: "Strip Plot", src: `{
  "data": { "values": [
    {"endpoint": "/login", "latency": 136},
    {"endpoint": "/login", "latency": 55},
    {"endpoint": "/login", "latency": 136},
    {"endpoint": "/login", "latency": 132},
    {"endpoint": "/login", "latency": 162},
    {"endpoint": "/login", "latency": 131},
    {"endpoint": "/login", "latency": 118},
    {"endpoint": "/login", "latency": 133},
    {"endpoint": "/login", "latency": 71},
    {"endpoint": "/login", "latency": 146},
    {"endpoint": "/login", "latency": 128},
    {"endpoint": "/login", "latency": 102},
    {"endpoint": "/search", "latency": 459},
    {"endpoint": "/search", "latency": 503},
    {"endpoint": "/search", "latency": 214},
    {"endpoint": "/search", "latency": 280},
    {"endpoint": "/search", "latency": 366},
    {"endpoint": "/search", "latency": 357},
    {"endpoint": "/search", "latency": 304},
    {"endpoint": "/search", "latency": 252},
    {"endpoint": "/search", "latency": 531},
    {"endpoint": "/search", "latency": 433},
    {"endpoint": "/search", "latency": 233},
    {"endpoint": "/search", "latency": 219},
    {"endpoint": "/checkout", "latency": 758},
    {"endpoint": "/checkout", "latency": 658},
    {"endpoint": "/checkout", "latency": 775},
    {"endpoint": "/checkout", "latency": 633},
    {"endpoint": "/checkout", "latency": 398},
    {"endpoint": "/checkout", "latency": 556},
    {"endpoint": "/checkout", "latency": 218},
    {"endpoint": "/checkout", "latency": 415},
    {"endpoint": "/checkout", "latency": 512},
    {"endpoint": "/checkout", "latency": 593},
    {"endpoint": "/checkout", "latency": 418},
    {"endpoint": "/checkout", "latency": 503},
    {"endpoint": "/profile", "latency": 87},
    {"endpoint": "/profile", "latency": 86},
    {"endpoint": "/profile", "latency": 90},
    {"endpoint": "/profile", "latency": 83},
    {"endpoint": "/profile", "latency": 75},
    {"endpoint": "/profile", "latency": 92},
    {"endpoint": "/profile", "latency": 81},
    {"endpoint": "/profile", "latency": 68},
    {"endpoint": "/profile", "latency": 71},
    {"endpoint": "/profile", "latency": 80},
    {"endpoint": "/profile", "latency": 78},
    {"endpoint": "/profile", "latency": 82}
  ]},
  "semantic_types": {
    "endpoint": "Category",
    "latency": {
      "semanticType": "Duration",
      "unit": "ms"
    }
  },
  "chart_spec": {
    "chartType": "Strip Plot",
    "title": "Latency by endpoint",
    "encodings": {
      "x": {
        "field": "endpoint"
      },
      "y": {
        "field": "latency"
      }
    }
  }
}`},
	{name: "ECDF", src: `{
  "data": { "values": [
    {"group": "Before", "score": 62},
    {"group": "Before", "score": 64},
    {"group": "Before", "score": 61},
    {"group": "Before", "score": 48},
    {"group": "Before", "score": 67},
    {"group": "Before", "score": 74},
    {"group": "Before", "score": 67},
    {"group": "Before", "score": 60},
    {"group": "Before", "score": 67},
    {"group": "Before", "score": 51},
    {"group": "Before", "score": 41},
    {"group": "Before", "score": 63},
    {"group": "Before", "score": 52},
    {"group": "Before", "score": 70},
    {"group": "Before", "score": 50},
    {"group": "Before", "score": 33},
    {"group": "Before", "score": 51},
    {"group": "Before", "score": 79},
    {"group": "Before", "score": 58},
    {"group": "Before", "score": 47},
    {"group": "Before", "score": 54},
    {"group": "Before", "score": 68},
    {"group": "Before", "score": 67},
    {"group": "Before", "score": 64},
    {"group": "Before", "score": 78},
    {"group": "Before", "score": 70},
    {"group": "Before", "score": 62},
    {"group": "Before", "score": 69},
    {"group": "Before", "score": 80},
    {"group": "Before", "score": 73},
    {"group": "Before", "score": 73},
    {"group": "Before", "score": 50},
    {"group": "Before", "score": 60},
    {"group": "Before", "score": 70},
    {"group": "Before", "score": 59},
    {"group": "Before", "score": 74},
    {"group": "Before", "score": 69},
    {"group": "Before", "score": 72},
    {"group": "Before", "score": 60},
    {"group": "Before", "score": 90},
    {"group": "Before", "score": 76},
    {"group": "Before", "score": 60},
    {"group": "Before", "score": 63},
    {"group": "Before", "score": 91},
    {"group": "Before", "score": 58},
    {"group": "Before", "score": 72},
    {"group": "Before", "score": 73},
    {"group": "Before", "score": 62},
    {"group": "Before", "score": 49},
    {"group": "Before", "score": 64},
    {"group": "Before", "score": 66},
    {"group": "Before", "score": 74},
    {"group": "Before", "score": 71},
    {"group": "Before", "score": 62},
    {"group": "Before", "score": 71},
    {"group": "Before", "score": 68},
    {"group": "Before", "score": 64},
    {"group": "Before", "score": 63},
    {"group": "Before", "score": 59},
    {"group": "Before", "score": 70},
    {"group": "After", "score": 59},
    {"group": "After", "score": 64},
    {"group": "After", "score": 71},
    {"group": "After", "score": 55},
    {"group": "After", "score": 66},
    {"group": "After", "score": 49},
    {"group": "After", "score": 63},
    {"group": "After", "score": 77},
    {"group": "After", "score": 77},
    {"group": "After", "score": 70},
    {"group": "After", "score": 68},
    {"group": "After", "score": 55},
    {"group": "After", "score": 91},
    {"group": "After", "score": 77},
    {"group": "After", "score": 83},
    {"group": "After", "score": 61},
    {"group": "After", "score": 69},
    {"group": "After", "score": 51},
    {"group": "After", "score": 80},
    {"group": "After", "score": 81},
    {"group": "After", "score": 50},
    {"group": "After", "score": 70},
    {"group": "After", "score": 78},
    {"group": "After", "score": 52},
    {"group": "After", "score": 51},
    {"group": "After", "score": 59},
    {"group": "After", "score": 64},
    {"group": "After", "score": 56},
    {"group": "After", "score": 71},
    {"group": "After", "score": 74},
    {"group": "After", "score": 78},
    {"group": "After", "score": 79},
    {"group": "After", "score": 88},
    {"group": "After", "score": 84},
    {"group": "After", "score": 57},
    {"group": "After", "score": 65},
    {"group": "After", "score": 59},
    {"group": "After", "score": 59},
    {"group": "After", "score": 70},
    {"group": "After", "score": 71},
    {"group": "After", "score": 76},
    {"group": "After", "score": 54},
    {"group": "After", "score": 57},
    {"group": "After", "score": 71},
    {"group": "After", "score": 69},
    {"group": "After", "score": 68},
    {"group": "After", "score": 70},
    {"group": "After", "score": 63},
    {"group": "After", "score": 79},
    {"group": "After", "score": 75},
    {"group": "After", "score": 70},
    {"group": "After", "score": 64},
    {"group": "After", "score": 69},
    {"group": "After", "score": 41},
    {"group": "After", "score": 60},
    {"group": "After", "score": 71},
    {"group": "After", "score": 54},
    {"group": "After", "score": 73},
    {"group": "After", "score": 73},
    {"group": "After", "score": 56}
  ]},
  "semantic_types": {
    "group": "Category",
    "score": "Score"
  },
  "chart_spec": {
    "chartType": "ECDF Plot",
    "title": "Exam scores before and after the course",
    "encodings": {
      "x": {
        "field": "score"
      },
      "color": {
        "field": "group"
      }
    }
  }
}`},
	{name: "Lollipop", src: `{
  "data": { "values": [
    {"country": "Qatar", "tonnes": 37.6},
    {"country": "Kuwait", "tonnes": 25.6},
    {"country": "Australia", "tonnes": 14.9},
    {"country": "United States", "tonnes": 14.4},
    {"country": "Canada", "tonnes": 14.2},
    {"country": "South Korea", "tonnes": 11.6},
    {"country": "Germany", "tonnes": 8.0},
    {"country": "China", "tonnes": 8.0},
    {"country": "Japan", "tonnes": 8.5},
    {"country": "United Kingdom", "tonnes": 4.7},
    {"country": "India", "tonnes": 2.0},
    {"country": "Brazil", "tonnes": 2.2}
  ]},
  "semantic_types": {
    "country": "Country",
    "tonnes": "Quantity"
  },
  "chart_spec": {
    "chartType": "Lollipop Chart",
    "title": "CO2 per person, 2022 (tonnes)",
    "encodings": {
      "x": {
        "field": "country",
        "sortBy": "y",
        "sortOrder": "descending"
      },
      "y": {
        "field": "tonnes"
      }
    }
  }
}`},
	{name: "Sparklines", src: `{
  "data": { "values": [
    {"month": "2024-01-01", "kpi": "Revenue ($k)", "value": 123.5},
    {"month": "2024-02-01", "kpi": "Revenue ($k)", "value": 120.2},
    {"month": "2024-03-01", "kpi": "Revenue ($k)", "value": 117.8},
    {"month": "2024-04-01", "kpi": "Revenue ($k)", "value": 116.7},
    {"month": "2024-05-01", "kpi": "Revenue ($k)", "value": 121.4},
    {"month": "2024-06-01", "kpi": "Revenue ($k)", "value": 120.8},
    {"month": "2024-07-01", "kpi": "Revenue ($k)", "value": 123.4},
    {"month": "2024-08-01", "kpi": "Revenue ($k)", "value": 119.4},
    {"month": "2024-09-01", "kpi": "Revenue ($k)", "value": 115.9},
    {"month": "2024-10-01", "kpi": "Revenue ($k)", "value": 114.9},
    {"month": "2024-11-01", "kpi": "Revenue ($k)", "value": 118.8},
    {"month": "2024-12-01", "kpi": "Revenue ($k)", "value": 122.9},
    {"month": "2025-01-01", "kpi": "Revenue ($k)", "value": 126.8},
    {"month": "2025-02-01", "kpi": "Revenue ($k)", "value": 126.1},
    {"month": "2025-03-01", "kpi": "Revenue ($k)", "value": 128.1},
    {"month": "2025-04-01", "kpi": "Revenue ($k)", "value": 129.5},
    {"month": "2025-05-01", "kpi": "Revenue ($k)", "value": 130.9},
    {"month": "2025-06-01", "kpi": "Revenue ($k)", "value": 128.1},
    {"month": "2025-07-01", "kpi": "Revenue ($k)", "value": 134.6},
    {"month": "2025-08-01", "kpi": "Revenue ($k)", "value": 132.8},
    {"month": "2025-09-01", "kpi": "Revenue ($k)", "value": 140.4},
    {"month": "2025-10-01", "kpi": "Revenue ($k)", "value": 147.4},
    {"month": "2025-11-01", "kpi": "Revenue ($k)", "value": 143.4},
    {"month": "2025-12-01", "kpi": "Revenue ($k)", "value": 144.7},
    {"month": "2024-01-01", "kpi": "Active users (k)", "value": 50.8},
    {"month": "2024-02-01", "kpi": "Active users (k)", "value": 54.5},
    {"month": "2024-03-01", "kpi": "Active users (k)", "value": 55.1},
    {"month": "2024-04-01", "kpi": "Active users (k)", "value": 54.6},
    {"month": "2024-05-01", "kpi": "Active users (k)", "value": 53.8},
    {"month": "2024-06-01", "kpi": "Active users (k)", "value": 57.4},
    {"month": "2024-07-01", "kpi": "Active users (k)", "value": 56.5},
    {"month": "2024-08-01", "kpi": "Active users (k)", "value": 57.9},
    {"month": "2024-09-01", "kpi": "Active users (k)", "value": 56.7},
    {"month": "2024-10-01", "kpi": "Active users (k)", "value": 57.7},
    {"month": "2024-11-01", "kpi": "Active users (k)", "value": 61.3},
    {"month": "2024-12-01", "kpi": "Active users (k)", "value": 60.0},
    {"month": "2025-01-01", "kpi": "Active users (k)", "value": 62.9},
    {"month": "2025-02-01", "kpi": "Active users (k)", "value": 63.8},
    {"month": "2025-03-01", "kpi": "Active users (k)", "value": 67.0},
    {"month": "2025-04-01", "kpi": "Active users (k)", "value": 69.1},
    {"month": "2025-05-01", "kpi": "Active users (k)", "value": 68.4},
    {"month": "2025-06-01", "kpi": "Active users (k)", "value": 71.7},
    {"month": "2025-07-01", "kpi": "Active users (k)", "value": 72.5},
    {"month": "2025-08-01", "kpi": "Active users (k)", "value": 70.6},
    {"month": "2025-09-01", "kpi": "Active users (k)", "value": 68.5},
    {"month": "2025-10-01", "kpi": "Active users (k)", "value": 69.4},
    {"month": "2025-11-01", "kpi": "Active users (k)", "value": 70.0},
    {"month": "2025-12-01", "kpi": "Active users (k)", "value": 69.7},
    {"month": "2024-01-01", "kpi": "Churn (%)", "value": 3.9},
    {"month": "2024-02-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2024-03-01", "kpi": "Churn (%)", "value": 3.7},
    {"month": "2024-04-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2024-05-01", "kpi": "Churn (%)", "value": 3.5},
    {"month": "2024-06-01", "kpi": "Churn (%)", "value": 3.6},
    {"month": "2024-07-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2024-08-01", "kpi": "Churn (%)", "value": 3.5},
    {"month": "2024-09-01", "kpi": "Churn (%)", "value": 3.7},
    {"month": "2024-10-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2024-11-01", "kpi": "Churn (%)", "value": 4.0},
    {"month": "2024-12-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2025-01-01", "kpi": "Churn (%)", "value": 3.7},
    {"month": "2025-02-01", "kpi": "Churn (%)", "value": 3.6},
    {"month": "2025-03-01", "kpi": "Churn (%)", "value": 3.9},
    {"month": "2025-04-01", "kpi": "Churn (%)", "value": 3.9},
    {"month": "2025-05-01", "kpi": "Churn (%)", "value": 3.8},
    {"month": "2025-06-01", "kpi": "Churn (%)", "value": 3.7},
    {"month": "2025-07-01", "kpi": "Churn (%)", "value": 3.5},
    {"month": "2025-08-01", "kpi": "Churn (%)", "value": 3.2},
    {"month": "2025-09-01", "kpi": "Churn (%)", "value": 2.9},
    {"month": "2025-10-01", "kpi": "Churn (%)", "value": 3.1},
    {"month": "2025-11-01", "kpi": "Churn (%)", "value": 2.9},
    {"month": "2025-12-01", "kpi": "Churn (%)", "value": 3.1}
  ]},
  "semantic_types": {
    "month": "Date",
    "kpi": "Category",
    "value": "Quantity"
  },
  "chart_spec": {
    "chartType": "Sparkline",
    "title": "KPIs, last 24 months",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "value"
      },
      "color": {
        "field": "kpi"
      }
    }
  }
}`},
	{name: "Connected Scatter", src: `{
  "data": { "values": [
    {"year": 2014, "unemployment": 6.2, "inflation": 1.6},
    {"year": 2015, "unemployment": 5.3, "inflation": 0.1},
    {"year": 2016, "unemployment": 4.9, "inflation": 1.3},
    {"year": 2017, "unemployment": 4.4, "inflation": 2.1},
    {"year": 2018, "unemployment": 3.9, "inflation": 2.4},
    {"year": 2019, "unemployment": 3.7, "inflation": 1.8},
    {"year": 2020, "unemployment": 8.1, "inflation": 1.2},
    {"year": 2021, "unemployment": 5.4, "inflation": 4.7},
    {"year": 2022, "unemployment": 3.6, "inflation": 8.0},
    {"year": 2023, "unemployment": 3.6, "inflation": 4.1},
    {"year": 2024, "unemployment": 4.0, "inflation": 2.9}
  ]},
  "semantic_types": {
    "year": "Year",
    "unemployment": {
      "semanticType": "Percentage",
      "unit": "%"
    },
    "inflation": {
      "semanticType": "Percentage",
      "unit": "%"
    }
  },
  "chart_spec": {
    "chartType": "Connected Scatter Plot",
    "title": "US unemployment vs inflation, 2014-2024",
    "encodings": {
      "x": {
        "field": "unemployment"
      },
      "y": {
        "field": "inflation"
      },
      "order": {
        "field": "year"
      }
    }
  }
}`},
	{name: "Ranged Dots", src: `{
  "data": { "values": [
    {"occupation": "Developers", "sex": "Men", "salary ($k)": 132},
    {"occupation": "Developers", "sex": "Women", "salary ($k)": 118},
    {"occupation": "Nurses", "sex": "Men", "salary ($k)": 81},
    {"occupation": "Nurses", "sex": "Women", "salary ($k)": 77},
    {"occupation": "Teachers", "sex": "Men", "salary ($k)": 64},
    {"occupation": "Teachers", "sex": "Women", "salary ($k)": 60},
    {"occupation": "Lawyers", "sex": "Men", "salary ($k)": 152},
    {"occupation": "Lawyers", "sex": "Women", "salary ($k)": 126},
    {"occupation": "Chefs", "sex": "Men", "salary ($k)": 52},
    {"occupation": "Chefs", "sex": "Women", "salary ($k)": 41},
    {"occupation": "Pharmacists", "sex": "Men", "salary ($k)": 136},
    {"occupation": "Pharmacists", "sex": "Women", "salary ($k)": 127}
  ]},
  "semantic_types": {
    "occupation": "Category",
    "sex": "Category",
    "salary ($k)": "Quantity"
  },
  "chart_spec": {
    "chartType": "Ranged Dot Plot",
    "title": "Median salary by occupation",
    "encodings": {
      "x": {
        "field": "salary ($k)"
      },
      "y": {
        "field": "occupation"
      },
      "color": {
        "field": "sex"
      }
    }
  }
}`},
	{name: "Area Chart", src: `{
  "data": { "values": [
    {"year": "1995", "online": 0.8},
    {"year": "1998", "online": 3.6},
    {"year": "2001", "online": 8.6},
    {"year": "2004", "online": 14.1},
    {"year": "2007", "online": 20.5},
    {"year": "2010", "online": 28.7},
    {"year": "2013", "online": 37.9},
    {"year": "2016", "online": 45.7},
    {"year": "2019", "online": 53.6},
    {"year": "2022", "online": 63.1},
    {"year": "2024", "online": 67.4}
  ]},
  "semantic_types": {
    "year": "Year",
    "online": {
      "semanticType": "Percentage",
      "unit": "%"
    }
  },
  "chart_spec": {
    "chartType": "Area Chart",
    "title": "Share of the world online",
    "encodings": {
      "x": {
        "field": "year"
      },
      "y": {
        "field": "online"
      }
    }
  }
}`},
	{name: "Diverging Heatmap", src: `{
  "data": { "values": [
    {"year": "2015", "month": "Jan", "anomaly": -0.86},
    {"year": "2015", "month": "Feb", "anomaly": -0.81},
    {"year": "2015", "month": "Mar", "anomaly": -0.68},
    {"year": "2015", "month": "Apr", "anomaly": -0.99},
    {"year": "2015", "month": "May", "anomaly": -0.99},
    {"year": "2015", "month": "Jun", "anomaly": -0.71},
    {"year": "2015", "month": "Jul", "anomaly": -0.85},
    {"year": "2015", "month": "Aug", "anomaly": -0.93},
    {"year": "2015", "month": "Sep", "anomaly": -1.02},
    {"year": "2015", "month": "Oct", "anomaly": -0.74},
    {"year": "2015", "month": "Nov", "anomaly": -0.59},
    {"year": "2015", "month": "Dec", "anomaly": -0.74},
    {"year": "2016", "month": "Jan", "anomaly": -0.36},
    {"year": "2016", "month": "Feb", "anomaly": -0.8},
    {"year": "2016", "month": "Mar", "anomaly": -0.35},
    {"year": "2016", "month": "Apr", "anomaly": -0.64},
    {"year": "2016", "month": "May", "anomaly": -0.55},
    {"year": "2016", "month": "Jun", "anomaly": -0.75},
    {"year": "2016", "month": "Jul", "anomaly": -1.1},
    {"year": "2016", "month": "Aug", "anomaly": -1.3},
    {"year": "2016", "month": "Sep", "anomaly": -0.65},
    {"year": "2016", "month": "Oct", "anomaly": -1.12},
    {"year": "2016", "month": "Nov", "anomaly": -0.75},
    {"year": "2016", "month": "Dec", "anomaly": -0.71},
    {"year": "2017", "month": "Jan", "anomaly": -0.49},
    {"year": "2017", "month": "Feb", "anomaly": -0.14},
    {"year": "2017", "month": "Mar", "anomaly": -0.01},
    {"year": "2017", "month": "Apr", "anomaly": -0.6},
    {"year": "2017", "month": "May", "anomaly": -0.45},
    {"year": "2017", "month": "Jun", "anomaly": -0.82},
    {"year": "2017", "month": "Jul", "anomaly": -0.74},
    {"year": "2017", "month": "Aug", "anomaly": -0.88},
    {"year": "2017", "month": "Sep", "anomaly": -1.01},
    {"year": "2017", "month": "Oct", "anomaly": -0.92},
    {"year": "2017", "month": "Nov", "anomaly": -0.76},
    {"year": "2017", "month": "Dec", "anomaly": -0.15},
    {"year": "2018", "month": "Jan", "anomaly": -0.18},
    {"year": "2018", "month": "Feb", "anomaly": -0.34},
    {"year": "2018", "month": "Mar", "anomaly": 0.11},
    {"year": "2018", "month": "Apr", "anomaly": 0.08},
    {"year": "2018", "month": "May", "anomaly": -0.43},
    {"year": "2018", "month": "Jun", "anomaly": -0.77},
    {"year": "2018", "month": "Jul", "anomaly": -0.82},
    {"year": "2018", "month": "Aug", "anomaly": -0.93},
    {"year": "2018", "month": "Sep", "anomaly": -0.72},
    {"year": "2018", "month": "Oct", "anomaly": -0.8},
    {"year": "2018", "month": "Nov", "anomaly": -0.57},
    {"year": "2018", "month": "Dec", "anomaly": -0.43},
    {"year": "2019", "month": "Jan", "anomaly": 0.05},
    {"year": "2019", "month": "Feb", "anomaly": 0.3},
    {"year": "2019", "month": "Mar", "anomaly": 0.17},
    {"year": "2019", "month": "Apr", "anomaly": -0.16},
    {"year": "2019", "month": "May", "anomaly": -0.28},
    {"year": "2019", "month": "Jun", "anomaly": -0.33},
    {"year": "2019", "month": "Jul", "anomaly": -0.52},
    {"year": "2019", "month": "Aug", "anomaly": -0.58},
    {"year": "2019", "month": "Sep", "anomaly": -0.74},
    {"year": "2019", "month": "Oct", "anomaly": -0.5},
    {"year": "2019", "month": "Nov", "anomaly": 0.11},
    {"year": "2019", "month": "Dec", "anomaly": -0.36},
    {"year": "2020", "month": "Jan", "anomaly": 0.17},
    {"year": "2020", "month": "Feb", "anomaly": 0.29},
    {"year": "2020", "month": "Mar", "anomaly": 0.42},
    {"year": "2020", "month": "Apr", "anomaly": -0.12},
    {"year": "2020", "month": "May", "anomaly": -0.21},
    {"year": "2020", "month": "Jun", "anomaly": -0.35},
    {"year": "2020", "month": "Jul", "anomaly": -0.34},
    {"year": "2020", "month": "Aug", "anomaly": -0.34},
    {"year": "2020", "month": "Sep", "anomaly": 0.05},
    {"year": "2020", "month": "Oct", "anomaly": 0.07},
    {"year": "2020", "month": "Nov", "anomaly": 0.21},
    {"year": "2020", "month": "Dec", "anomaly": -0.26},
    {"year": "2021", "month": "Jan", "anomaly": 0.01},
    {"year": "2021", "month": "Feb", "anomaly": 0.52},
    {"year": "2021", "month": "Mar", "anomaly": 0.61},
    {"year": "2021", "month": "Apr", "anomaly": 0.23},
    {"year": "2021", "month": "May", "anomaly": 0.18},
    {"year": "2021", "month": "Jun", "anomaly": -0.35},
    {"year": "2021", "month": "Jul", "anomaly": -0.17},
    {"year": "2021", "month": "Aug", "anomaly": 0.17},
    {"year": "2021", "month": "Sep", "anomaly": 0.13},
    {"year": "2021", "month": "Oct", "anomaly": 0.24},
    {"year": "2021", "month": "Nov", "anomaly": 0.45},
    {"year": "2021", "month": "Dec", "anomaly": 0.07},
    {"year": "2022", "month": "Jan", "anomaly": 0.23},
    {"year": "2022", "month": "Feb", "anomaly": 0.3},
    {"year": "2022", "month": "Mar", "anomaly": 0.52},
    {"year": "2022", "month": "Apr", "anomaly": 0.54},
    {"year": "2022", "month": "May", "anomaly": 0.6},
    {"year": "2022", "month": "Jun", "anomaly": 0.32},
    {"year": "2022", "month": "Jul", "anomaly": 0.18},
    {"year": "2022", "month": "Aug", "anomaly": 0.23},
    {"year": "2022", "month": "Sep", "anomaly": 0.04},
    {"year": "2022", "month": "Oct", "anomaly": 0.2},
    {"year": "2022", "month": "Nov", "anomaly": -0.03},
    {"year": "2022", "month": "Dec", "anomaly": 0.61},
    {"year": "2023", "month": "Jan", "anomaly": 0.49},
    {"year": "2023", "month": "Feb", "anomaly": 1.0},
    {"year": "2023", "month": "Mar", "anomaly": 0.78},
    {"year": "2023", "month": "Apr", "anomaly": 0.45},
    {"year": "2023", "month": "May", "anomaly": 0.2},
    {"year": "2023", "month": "Jun", "anomaly": 0.16},
    {"year": "2023", "month": "Jul", "anomaly": 0.34},
    {"year": "2023", "month": "Aug", "anomaly": 0.35},
    {"year": "2023", "month": "Sep", "anomaly": -0.03},
    {"year": "2023", "month": "Oct", "anomaly": 0.03},
    {"year": "2023", "month": "Nov", "anomaly": 0.48},
    {"year": "2023", "month": "Dec", "anomaly": 0.64},
    {"year": "2024", "month": "Jan", "anomaly": 0.77},
    {"year": "2024", "month": "Feb", "anomaly": 0.69},
    {"year": "2024", "month": "Mar", "anomaly": 0.92},
    {"year": "2024", "month": "Apr", "anomaly": 0.41},
    {"year": "2024", "month": "May", "anomaly": 0.49},
    {"year": "2024", "month": "Jun", "anomaly": 0.48},
    {"year": "2024", "month": "Jul", "anomaly": 0.73},
    {"year": "2024", "month": "Aug", "anomaly": 0.48},
    {"year": "2024", "month": "Sep", "anomaly": 0.68},
    {"year": "2024", "month": "Oct", "anomaly": 0.49},
    {"year": "2024", "month": "Nov", "anomaly": 0.44},
    {"year": "2024", "month": "Dec", "anomaly": 0.58}
  ]},
  "semantic_types": {
    "year": "Year",
    "month": "Month",
    "anomaly": {
      "semanticType": "Temperature",
      "unit": "°C"
    }
  },
  "chart_spec": {
    "chartType": "Heatmap",
    "title": "Temperature anomaly vs 1991-2020 (°C)",
    "encodings": {
      "x": {
        "field": "month"
      },
      "y": {
        "field": "year"
      },
      "color": {
        "field": "anomaly"
      }
    }
  }
}`},
}
