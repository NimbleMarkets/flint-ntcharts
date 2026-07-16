package main

import "testing"

func TestParseConfig(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(cfg config) bool
	}{
		{"file arg", []string{"chart.json"}, false, func(c config) bool { return c.file == "chart.json" }},
		{"stdin", []string{"--stdin"}, false, func(c config) bool { return c.stdin }},
		{"socket", []string{"--listen", "/tmp/f.sock"}, false, func(c config) bool { return c.socket == "/tmp/f.sock" }},
		{"combined", []string{"--stdin", "chart.json"}, false, func(c config) bool { return c.stdin && c.file == "chart.json" }},
		{"no sources", []string{}, true, nil},
		{"two positionals", []string{"a.json", "b.json"}, true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := parseConfig(c.args)
			if c.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && !c.check(cfg) {
				t.Fatalf("config check failed: %+v", cfg)
			}
		})
	}
}
