package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func framedLines(t *testing.T, s spec.Spec, w, h int) []string {
	t.Helper()
	v, err := FramedView(s, w, h)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(ansi.Strip(v), "\n")
}

func stackedBars() spec.Spec {
	return spec.Spec{Type: spec.ChartTypeBar, Title: "Units by device", Subtitle: "2026",
		XAxis: spec.XAxis{Title: "device", Labels: []string{"Laptop", "Phone"}}, YAxis: spec.YAxis{Title: "units"},
		Options: spec.Options{Stacked: true, ShowLegend: true},
		Data: spec.Data{Series: []spec.Series{
			{Name: "USA", Color: "#4e79a7", Values: []spec.DataPoint{{Y: 3}, {Y: 5}}},
			{Name: "China", Color: "#f28e2c", Values: []spec.DataPoint{{Y: 2}, {Y: 4}}},
		}}}
}

func TestFramedViewNamesTitleLegendAndAxes(t *testing.T) {
	lines := framedLines(t, stackedBars(), 40, 16)
	if len(lines) != 16 {
		t.Fatalf("%d rows, want the 16 asked for", len(lines))
	}
	if lines[0] != "Units by device · 2026" {
		t.Fatalf("title row = %q", lines[0])
	}
	// With room, the legend ends where the plot does, on its own row, and
	// the Y title's row keeps it clear of the plot.
	if strings.TrimSpace(lines[1]) != "■ USA   ■ China" || ansi.StringWidth(lines[1]) != 40 {
		t.Fatalf("legend row = %q", lines[1])
	}
	if strings.TrimSpace(lines[2]) != "units" {
		t.Fatalf("Y title row = %q", lines[2])
	}
	if strings.TrimSpace(lines[15]) != "device" {
		t.Fatalf("X title row = %q", lines[15])
	}
}

func TestFramedViewSharesTheLegendRowWhenTight(t *testing.T) {
	lines := framedLines(t, stackedBars(), 40, 12)
	if !strings.HasPrefix(lines[1], "units") || !strings.HasSuffix(lines[1], "■ USA   ■ China") {
		t.Fatalf("Y title and legend row = %q", lines[1])
	}
}

func TestFramedViewLegendPosition(t *testing.T) {
	s := stackedBars()
	s.Options.LegendPosition = spec.LegendBottom
	lines := framedLines(t, s, 40, 16)
	if !strings.Contains(lines[len(lines)-1], "■ USA") {
		t.Fatalf("legend not at the bottom: %q", lines[len(lines)-1])
	}
	s.Options.LegendPosition = spec.LegendNone
	if v := strings.Join(framedLines(t, s, 40, 16), "\n"); strings.Contains(v, "■") {
		t.Fatalf("legend drawn with position none:\n%s", v)
	}
}

func TestFramedViewLeavesASmallPlotAlone(t *testing.T) {
	lines := framedLines(t, stackedBars(), 40, 5)
	if lines[0] == "Units by device · 2026" {
		t.Fatal("a 5-row plot must not give rows to chrome")
	}
}

func TestFramedViewShowsAHeatmapRamp(t *testing.T) {
	s := spec.Spec{Type: spec.ChartTypeHeatmap, Theme: spec.Theme{Gradient: []string{"#000000", "#ffffff"}},
		XAxis: spec.XAxis{Labels: []string{"a", "b"}}, YAxis: spec.YAxis{Labels: []string{"r"}},
		Heat: &spec.HeatData{Cells: []spec.HeatCell{{X: 0, Y: 0, Z: 12}, {X: 1, Y: 0, Z: 98}}}}
	lines := framedLines(t, s, 40, 12)
	if ramp := strings.TrimSpace(lines[0]); !strings.HasPrefix(ramp, "12 ■") || !strings.HasSuffix(lines[0], "■ 98") {
		t.Fatalf("ramp row = %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "" {
		t.Fatalf("the ramp must keep a row clear of the plot: %q", lines[1])
	}
}
