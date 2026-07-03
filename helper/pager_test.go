package helper

import "testing"

func TestLineCount(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{input: "", want: 0},
		{input: "one", want: 1},
		{input: "one\n", want: 1},
		{input: "one\ntwo", want: 2},
		{input: "one\ntwo\n", want: 2},
		{input: "a\nb\nc\n", want: 3},
	}

	for _, tt := range tests {
		if got := lineCount(tt.input); got != tt.want {
			t.Fatalf("lineCount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPagerCommandUsesEnv(t *testing.T) {
	t.Setenv("PAGER", "less -FRX")

	name, args := pagerCommand()
	if name != "less" || len(args) != 1 || args[0] != "-FRX" {
		t.Fatalf("pagerCommand() = (%q, %v), want (less, [-FRX])", name, args)
	}
}

func TestPagerCommandDefault(t *testing.T) {
	t.Setenv("PAGER", "")

	name, args := pagerCommand()
	if name != "less" || len(args) != 1 || args[0] != "-R" {
		t.Fatalf("pagerCommand() = (%q, %v), want (less, [-R])", name, args)
	}
}
