package cli

import (
	"bytes"
	"flag"
	"slices"
	"strings"
	"testing"
)

func TestParseArgsFlagsAnywhere(t *testing.T) {
	for _, args := range [][]string{
		{"--package", "a.b", "app"},
		{"app", "--package", "a.b"},
	} {
		fs := NewFlagSet("t")
		pkg := fs.String("package", "", "")
		pos, err := ParseArgs(fs, args)
		if err != nil || *pkg != "a.b" || !slices.Equal(pos, []string{"app"}) {
			t.Errorf("ParseArgs(%q) = %q, package=%q, err=%v", args, pos, *pkg, err)
		}
	}
}

func TestRunExitCodes(t *testing.T) {
	ok := &Command{Name: "ok", Run: func(*Context, []string) error { return nil }}
	bad := &Command{Name: "bad", Usage: "bad", Run: func(*Context, []string) error { return Usagef("плохо") }}
	help := &Command{Name: "help-me", Usage: "help-me", Help: "справка", Run: func(*Context, []string) error { return flag.ErrHelp }}
	commands := []*Command{ok, bad, help}

	cases := []struct {
		args []string
		code int
		out  string
	}{
		{nil, 0, "Команды:"},
		{[]string{"ok"}, 0, ""},
		{[]string{"bad"}, 2, "плохо"},
		{[]string{"help-me"}, 0, "справка"},
		{[]string{"nope"}, 2, "неизвестная команда"},
	}
	for _, c := range cases {
		var out bytes.Buffer
		code := Run(&Context{Stdout: &out, Stderr: &out}, c.args, commands)
		if code != c.code || !strings.Contains(out.String(), c.out) {
			t.Errorf("Run(%q) = %d, вывод %q; ожидали %d и %q", c.args, code, out.String(), c.code, c.out)
		}
	}
}
