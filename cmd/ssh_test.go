package cmd

import (
	"reflect"
	"testing"
)

func TestBuildExecRemoteCommandQuotesArgs(t *testing.T) {
	got := buildExecRemoteCommand([]string{"printf", `%s\n`, "hello world", "it's ok"}, false)
	want := []string{`'printf' '%s\n' 'hello world' 'it'\''s ok'`}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildExecRemoteCommand() = %#v, want %#v", got, want)
	}
}

func TestBuildExecRemoteCommandShellMode(t *testing.T) {
	got := buildExecRemoteCommand([]string{"cd /tmp &&", "pwd"}, true)
	want := []string{`sh -lc 'cd /tmp && pwd'`}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildExecRemoteCommand(shell) = %#v, want %#v", got, want)
	}
}
