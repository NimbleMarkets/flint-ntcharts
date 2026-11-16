package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// FramedView draws a chart the way a reader needs it in a terminal: its title,
// the names of its colours, and what its axes measure, around the plot.
// ntcharts' spec.Build draws the plot alone, so the frame takes the rows it
// needs from the height and builds the plot in the rest:
//
//	Title · subtitle                       (when the spec has a title)
//	revenue              ■ USA ■ China     (Y axis title; legend on top)
//	<plot>
//	                              device   (X axis title, with room)
//
// The legend names each series in its colour (■ for bars and areas, ─ for
// lines, the series' symbol for points) or, for a heatmap, shows the ramp
// from its lowest value to its highest. Options.LegendPosition moves it
// below the plot or turns it off. FramedView depends only on the spec, so it can
// move into ntcharts as is.
func FramedView(s spec.Spec, w, h int) (string, error) {
	if w < 1 || h < 1 {
		return "", fmt.Errorf("frame: bad size %dx%d", w, h)
	}
	faint := lipgloss.NewStyle().Faint(true)
	var top, bottom []string

	if s.Title != "" {
		title := lipgloss.NewStyle().Bold(true).Render(ansi.Truncate(s.Title, w, "…"))
		if s.Subtitle != "" && ansi.StringWidth(s.Title)+3 < w {
			title += faint.Render(ansi.Truncate(" · "+s.Subtitle, w-ansi.StringWidth(s.Title), "…"))
		}
		top = append(top, title)
	}

	legend := legendRows(s, w)
	position := s.Options.LegendPosition
	if position == "" {
		position = spec.LegendTop
	}

	// What the axes measure. A horizontal bar chart measures across, so its
	// value title goes under the plot and its category title above.
	across, up := s.XAxis.Title, s.YAxis.Title
	if s.Type == spec.ChartTypeBar && s.Options.Orientation == spec.OrientationHorizontal {
		across, up = up, across
	}
	if s.Type == spec.ChartTypeSparkline || s.Type == spec.ChartTypeHeatmap {
		up = ""
	}

	// Rows the plot cannot do without: horizontal bars need two each.
	need := 4
	if s.Type == spec.ChartTypeBar && s.Options.Orientation == spec.OrientationHorizontal {
		n := len(s.XAxis.Labels)
		if n == 0 {
			n = len(spec.DeriveBarLabels(s.Data))
		}
		need = max(need, 2*n-1)
	}

	// Lay the chrome out, then give it up a piece at a time until the plot
	// has the rows it needs: the room around the legend, the X title, the
	// Y title, the title, the legend.
	title := top
	for drop := 0; ; drop++ {
		roomy := drop < 1
		top, bottom = nil, nil
		if drop < 4 {
			top = append(top, title...)
		}
		upHere := up
		if drop >= 3 {
			upHere = ""
		}
		if drop < 5 && len(legend) > 0 {
			switch position {
			case spec.LegendTop:
				rows := rightAligned(legend, w)
				if !roomy && upHere != "" && ansi.StringWidth(upHere)+2+ansi.StringWidth(legend[0]) <= w {
					// Tight: the Y title shares the legend's first row.
					rows[0] = faint.Render(upHere) + strings.Repeat(" ", w-ansi.StringWidth(upHere)-ansi.StringWidth(legend[0])) + legend[0]
					upHere = ""
				}
				top = append(top, rows...)
				// Roomy: the legend keeps a row clear of the plot, which
				// the Y title's row provides on the right.
				if roomy && upHere == "" {
					top = append(top, "")
				}
			case spec.LegendBottom:
				if roomy {
					bottom = append(bottom, "")
				}
				bottom = append(bottom, rightAligned(legend, w)...)
			}
		}
		if upHere != "" {
			top = append(top, faint.Render(ansi.Truncate(upHere, w, "…")))
		}
		// The X title only when the plot keeps most of the height.
		plot := h - len(top) - len(bottom)
		if drop < 2 && across != "" && s.Type != spec.ChartTypeSparkline && plot-1 >= 10 {
			bottom = append([]string{strings.Repeat(" ", max(0, w-ansi.StringWidth(across))) + faint.Render(ansi.Truncate(across, w, "…"))}, bottom...)
		}
		plot = h - len(top) - len(bottom)
		if roomy && len(legend) > 0 && plot < max(need, 10) {
			continue // room around the legend only beside a roomy plot
		}
		if plot >= need || drop >= 5 {
			break
		}
	}
	plot := h - len(top) - len(bottom)
	if plot < 4 { // too small for chrome: the plot alone
		top, bottom, plot = nil, nil, h
	}

	s.Width, s.Height = w, plot
	model, err := spec.Build(s)
	if err != nil {
		return "", err
	}
	v, ok := model.(interface{ View() string })
	if !ok {
		return "", fmt.Errorf("chart model %T has no View", model)
	}
	rows := append(append(top, v.View()), bottom...)
	return strings.Join(rows, "\n"), nil
}

// rightAligned pads each row on the left to w columns, so the legend ends
// where the plot does.
func rightAligned(rows []string, w int) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.Repeat(" ", max(0, w-ansi.StringWidth(r))) + r
	}
	return out
}

// legendRows lays the legend out in at most two rows of w columns; entries
// that do not fit are counted in a final "+N".
func legendRows(s spec.Spec, w int) []string {
	if s.Options.LegendPosition == spec.LegendNone {
		return nil
	}
	if s.Type == spec.ChartTypeHeatmap {
		if ramp := heatRamp(s, w); ramp != "" {
			return []string{ramp}
		}
		return nil
	}
	series := s.Data.Series
	if len(series) < 2 && !s.Options.ShowLegend {
		return nil
	}
	// A name appears once, as its last series draws it (a regression's
	// points after its fitted curve).
	last := map[string]int{}
	for i, ser := range series {
		last[ser.Name] = i
	}
	var entries []string
	for i, ser := range series {
		if ser.Name == "" || strings.HasPrefix(ser.Name, "__") || last[ser.Name] != i {
			continue
		}
		mark := "■"
		switch {
		case s.Type == spec.ChartTypeScatter && ser.Symbol != "":
			mark = ser.Symbol
		case s.Type == spec.ChartTypeScatter:
			mark = "•"
		case ser.Type == "line" || s.Type == spec.ChartTypeLine || s.Type == spec.ChartTypeTimeSeries:
			mark = "─"
		}
		style := lipgloss.NewStyle()
		if c := seriesColour(s, ser, i); c != "" {
			style = style.Foreground(lipgloss.Color(c))
		}
		entries = append(entries, style.Render(mark)+" "+ansi.Truncate(ser.Name, max(4, w/3), "…"))
	}
	if len(entries) == 0 {
		return nil
	}
	var rows []string
	row := ""
	for i, e := range entries {
		next := e
		if row != "" {
			next = row + "   " + e
		}
		if ansi.StringWidth(next) <= w {
			row = next
			continue
		}
		rows = append(rows, row)
		row = e
		if len(rows) == 2 {
			more := fmt.Sprintf("+%d", len(entries)-i)
			last := rows[1]
			for ansi.StringWidth(last)+3+len(more) > w && strings.Contains(last, "   ") {
				last = last[:strings.LastIndex(last, "   ")]
			}
			rows[1] = last + "   " + more
			return rows
		}
	}
	return append(rows, row)
}

// seriesColour is the colour ntcharts draws a series in: its own, else the
// theme palette's.
func seriesColour(s spec.Spec, ser spec.Series, i int) string {
	if ser.Color != "" {
		return ser.Color
	}
	if i < len(s.Theme.Palette) {
		return s.Theme.Palette[i]
	}
	return ""
}

// heatRamp shows a heatmap's colours from its lowest value to its highest:
// "12 ████████ 98".
func heatRamp(s spec.Spec, w int) string {
	stops := s.Theme.Gradient
	if len(stops) < 2 || s.Heat == nil {
		return ""
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, c := range s.Heat.Cells {
		lo, hi = math.Min(lo, c.Z), math.Max(hi, c.Z)
	}
	for _, row := range s.Heat.Matrix {
		for _, z := range row {
			lo, hi = math.Min(lo, z), math.Max(hi, z)
		}
	}
	if s.Heat.MinValue != nil {
		lo = *s.Heat.MinValue
	}
	if s.Heat.MaxValue != nil {
		hi = *s.Heat.MaxValue
	}
	if math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return ""
	}
	label := func(v float64) string { return strconv.FormatFloat(v, 'g', 3, 64) }
	cells := min(16, w-ansi.StringWidth(label(lo))-ansi.StringWidth(label(hi))-2)
	if cells < 4 {
		return ""
	}
	var b strings.Builder
	b.WriteString(label(lo) + " ")
	for i := 0; i < cells; i++ {
		t := float64(i) / float64(cells-1)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(stops[int(math.Round(t*float64(len(stops)-1)))])).Render("■"))
	}
	b.WriteString(" " + label(hi))
	return b.String()
}
