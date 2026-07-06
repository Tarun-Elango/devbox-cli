package main

import "testing"

func TestResolveHelpTopic(t *testing.T) {
	tests := []struct {
		command      string
		args         []string
		wantTopic    string
		wantGeneral  bool
		wantOK       bool
	}{
		{"help", nil, "", true, true},
		{"help", []string{"box"}, "box", false, true},
		{"help", []string{"-h"}, "", true, true},
		{"box", []string{"help"}, "box", false, true},
		{"box", []string{"-h"}, "box", false, true},
		{"ls", []string{"help"}, "box", false, true},
		{"snapshot", []string{"help"}, "snapshot", false, true},
		{"ssh", []string{"help"}, "ssh", false, true},
		{"snapshot", []string{"create", "help"}, "snapshot", false, true},
		{"create", []string{"mybox", "help"}, "create", false, true},
		{"ssh", []string{"mybox", "-h"}, "ssh", false, true},
		{"ls", []string{"mybox"}, "", false, false},
	}

	for _, tt := range tests {
		topic, general, ok := resolveHelpTopic(tt.command, tt.args)
		if ok != tt.wantOK || general != tt.wantGeneral || topic != tt.wantTopic {
			t.Errorf("resolveHelpTopic(%q, %v) = (%q, %v, %v), want (%q, %v, %v)",
				tt.command, tt.args, topic, general, ok, tt.wantTopic, tt.wantGeneral, tt.wantOK)
		}
	}
}
