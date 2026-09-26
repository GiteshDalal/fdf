package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/GiteshDalal/fdf/cli/internal/install"
)

func mustTopic(t *testing.T, name string) helpTopic {
	t.Helper()
	topic, ok := findTopic(name)
	if !ok {
		t.Fatalf("no help topic for %q", name)
	}
	return topic
}

// The topics are the one source of the command list: every command fdf
// dispatches has exactly one, with a line in the overview, and every topic is
// a command.
func TestEveryCommandHasOneTopicAndOverviewLine(t *testing.T) {
	for name := range commands {
		n := 0
		for _, topic := range helpTopics {
			if topic.name == name {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: %d help topics, want 1", name, n)
			continue
		}
		topic := mustTopic(t, name)
		if topic.group == "" || topic.summary == "" {
			t.Errorf("%s: its topic needs a group and a summary for the overview", name)
		}
		if line := fmt.Sprintf("\n  %-10s %s\n", name, topic.summary); !strings.Contains(overview(), line) {
			t.Errorf("the overview has no line for %s: %q", name, line)
		}
	}
	for _, topic := range helpTopics {
		if commands[topic.name] == nil {
			t.Errorf("help topic %q is not a command", topic.name)
		}
	}
	// A group's commands sit together, so the overview lists each group once.
	seen := map[string]bool{}
	for i, topic := range helpTopics {
		if i > 0 && topic.group != helpTopics[i-1].group && seen[topic.group] {
			t.Errorf("group %q is split in two", topic.group)
		}
		seen[topic.group] = true
	}
}

// Every flag a command defines is in its topic's flag table, and every flag
// the table lists is one the command defines.
func TestEveryFlagIsInItsTopic(t *testing.T) {
	var sets []*flag.FlagSet
	flagSetHook = func(fs *flag.FlagSet) { sets = append(sets, fs) }
	defer func() { flagSetHook = nil }()
	for name, cmd := range commands {
		if code := cmd([]string{"-h"}, io.Discard); code != 0 {
			t.Errorf("fdf %s -h: exit %d", name, code)
		}
	}
	// bug defines --no-log only to refuse it: a cleared bug is always logged.
	refusedOnly := map[string]bool{"bug --no-log": true}
	flagOf := func(doc string) string { return strings.TrimLeft(strings.Fields(doc)[0], "-") }
	for _, fs := range sets {
		topic := mustTopic(t, fs.Name())
		listed := map[string]bool{}
		for _, f := range topic.flags {
			listed[flagOf(f.flag)] = true
			for _, word := range strings.Fields(strings.NewReplacer(",", " ", "(", " ", ")", " ").Replace(f.text)) {
				if strings.HasPrefix(word, "-") {
					listed[strings.TrimLeft(word, "-")] = true
				}
			}
		}
		fs.VisitAll(func(f *flag.Flag) {
			if !listed[f.Name] && !refusedOnly[fs.Name()+" --"+f.Name] {
				t.Errorf("fdf %s defines -%s, but its help topic does not list it", fs.Name(), f.Name)
			}
		})
		for _, f := range topic.flags {
			if fs.Lookup(flagOf(f.flag)) == nil {
				t.Errorf("fdf %s's help lists %s, which it does not define", fs.Name(), f.flag)
			}
		}
	}
}

// Help text is wrapped: a topic body at 74 columns, and everything fdf help
// prints at 80.
func TestHelpFitsItsWidth(t *testing.T) {
	for _, topic := range helpTopics {
		for _, line := range strings.Split(topic.body, "\n") {
			if n := utf8.RuneCountInString(line); n > 74 {
				t.Errorf("%s: body line of %d runes: %q", topic.name, n, line)
			}
		}
	}
	var out bytes.Buffer
	runHelp(nil, &out)
	for name, text := range map[string]string{"fdf help": out.String(), "the overview": overview()} {
		for _, line := range strings.Split(text, "\n") {
			if n := utf8.RuneCountInString(line); n > helpWidth {
				t.Errorf("%s: line of %d runes: %q", name, n, line)
			}
			if strings.TrimRight(line, " ") != line {
				t.Errorf("%s: trailing space: %q", name, line)
			}
		}
	}
}

// People ask fdf for a skill by name; the answer is what the skill is for, and
// that the agent runs it.
func TestHelpExplainsASkillName(t *testing.T) {
	want := `fdf help: "plan" is not a command. fdf-plan is a skill: it turns an approved
spec into tasks and slug.test.md. Skills are run by your AI agent, not by
fdf — ` + "`fdf install <harness>`" + ` installs them; ask the agent to use fdf-plan.
commands: init, install, migrate, new, adopt, change, fix, history, bug, debt,
practice, validate, log, mv, lexicon, release, spec, serve, help, version
`
	var out bytes.Buffer
	if code := runHelp([]string{"plan"}, &out); code != 2 || out.String() != want {
		t.Errorf("fdf help plan: exit %d\n got: %q\nwant: %q", code, out.String(), want)
	}
	out.Reset()
	runHelp([]string{"fdf-plan"}, &out)
	if !strings.Contains(out.String(), `"fdf-plan" is not a command. fdf-plan is a skill`) {
		t.Errorf("fdf help fdf-plan:\n%s", out.String())
	}
	out.Reset()
	runHelp([]string{"fdf-init"}, &out)
	if !strings.Contains(out.String(), "For the init command, see `fdf help init`.") {
		t.Errorf("a skill named like a command points at the command too:\n%s", out.String())
	}
	if code, _, errOut := fdfRun("plan"); code != 2 || !strings.HasPrefix(errOut, `fdf: "plan" is not a command. fdf-plan is a skill`) {
		t.Errorf("fdf plan: exit %d\n%s", code, errOut)
	}
}

func TestUnknownCommandSuggestsANearOne(t *testing.T) {
	for name, want := range map[string]string{"lexcon": "lexicon", "valdate": "validate", "debts": "debt", "fx": "fix"} {
		code, _, errOut := fdfRun(name)
		if code != 2 || !strings.Contains(errOut, fmt.Sprintf("unknown command %q — did you mean %q?", name, want)) {
			t.Errorf("fdf %s: exit %d\n%s", name, code, errOut)
		}
	}
	// A short name is not "near" every other short one.
	if _, _, errOut := fdfRun("go"); strings.Contains(errOut, "did you mean") {
		t.Errorf("fdf go should not guess:\n%s", errOut)
	}
	if _, _, errOut := fdfRun("--root", "x", "validate"); !strings.Contains(errOut, "the command comes first, then its flags") {
		t.Errorf("a flag before the command:\n%s", errOut)
	}
}

// The skills the help names are the skills install places, and the install
// topic counts them.
func TestHelpSkillsMatchInstall(t *testing.T) {
	var names []string
	for name := range skills {
		names = append(names, name)
	}
	installed := install.SkillNames()
	sort.Strings(names)
	sort.Strings(installed)
	if strings.Join(names, ",") != strings.Join(installed, ",") {
		t.Errorf("help knows skills %v; install places %v", names, installed)
	}
	if body := mustTopic(t, "install").body; !strings.Contains(body, " "+countWord(len(installed))+" FDF skills") {
		t.Errorf("the install topic should count %d skills:\n%s", len(installed), body)
	}
}

func TestWrapUsageBreaksBetweenParts(t *testing.T) {
	got := wrapUsage("usage: ", mustTopic(t, "debt").usage)
	want := "usage: fdf debt [--root <dir>] [--open|--accepted|--resolved]\n" +
		"                [--cleanup [--dry-run] [--no-log]] [--resource <paths>]\n" +
		"                [[<group>/…]<slug>]"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
