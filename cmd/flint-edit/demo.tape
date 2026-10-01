# flint-edit demo: edit a flint spec on the left, watch the terminal chart on
# the right. Run from the repo root:
#
#   task gif        (builds bin/flint-edit first)
#   # or: vhs cmd/flint-edit/demo.tape
Output cmd/flint-edit/demo.gif

Require echo

Set Shell "bash"
Set Padding 0
Set Framerate 30
Set FontSize 16
Set Width 1400
Set Height 640
Set TypingSpeed 120ms

Hide
Type "./bin/flint-edit"
Enter
Sleep 1.5s
Show

# Example 1 (bar chart) is loaded at startup.
Sleep 2s

# Live edit: walk up to the "May" row and change its revenue. The chart
# recompiles on every keystroke.
Up@120ms 7
Right@25ms 35
Sleep 500ms
Backspace@200ms 3
Type "320"
Sleep 2.5s

# ctrl+n steps through the built-in examples: a time series, candlesticks,
# a log-scale line, a heatmap, a histogram, then a calendar heatmap.
Ctrl+N
Sleep 2.5s
Ctrl+N
Sleep 2.5s
Ctrl+N
Sleep 2.5s
Ctrl+N
Sleep 2.5s
Ctrl+N
Sleep 2.5s
Ctrl+N
Sleep 3s

Hide
Ctrl+C
