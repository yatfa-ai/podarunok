// Command roleplay launches `claude` with a role persona injected via
// --append-system-prompt. Personas live as plain markdown in
// ~/.config/roleplay/roles/<role>.md. Memory is Claude Code's native machinery
// (CLAUDE.md + ~/.claude) — roleplay intentionally does NOT run its own store.
//
// --dangerously-skip-permissions is per-host configurable (config.json) and
// overridable per-invocation (flag / env), never hardcoded.
//
// Usage:
//
//	roleplay <role> [workdir] [claude-flags...]   start a session in that role
//	roleplay list                                 list roles
//	roleplay new <role>                           scaffold a new role
//
// roleplay-specific flags (filtered out, never passed to claude):
//
//	--skip-permissions / --no-skip-permissions
//	env ROLEPLAY_SKIP=1|0
//	env ROLEPLAY_DRY=1   print the claude command instead of running it
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func configDir() string {
	if d := os.Getenv("ROLEPLAY_CONFIG"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fail("home dir: %v", err)
	}
	return filepath.Join(home, ".config", "roleplay")
}

// config is ~/.config/roleplay/config.json. SkipHosts is a list of hostnames on
// which --dangerously-skip-permissions is on by default.
type config struct {
	SkipPermissions bool     `json:"skip_permissions"`
	SkipHosts       []string `json:"skip_hosts"`
}

func loadConfig() config {
	var c config
	if b, err := os.ReadFile(filepath.Join(configDir(), "config.json")); err == nil {
		_ = json.Unmarshal(b, &c) // malformed config ⇒ silent defaults
	}
	return c
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage()
		return
	}
	switch args[0] {
	case "list":
		listRoles()
	case "new":
		if len(args) < 2 {
			fail("usage: roleplay new <role>")
		}
		scaffold(args[1])
	default:
		run(args)
	}
}

// run is the main path: <role> [workdir] [flags...].
func run(args []string) {
	role := args[0]
	rest := args[1:]

	// Pull roleplay's own flags out; everything else is passed through to claude.
	var skipFlag *bool
	claudeArgs := make([]string, 0, len(rest))
	for _, a := range rest {
		switch a {
		case "--skip-permissions":
			t := true
			skipFlag = &t
		case "--no-skip-permissions":
			f := false
			skipFlag = &f
		default:
			claudeArgs = append(claudeArgs, a)
		}
	}

	// Optional workdir: a leading arg that is an existing directory → cd into it.
	if len(claudeArgs) > 0 {
		if fi, err := os.Stat(claudeArgs[0]); err == nil && fi.IsDir() {
			if err := os.Chdir(claudeArgs[0]); err != nil {
				fail("cd %s: %v", claudeArgs[0], err)
			}
			claudeArgs = claudeArgs[1:]
		}
	}

	persona, err := os.ReadFile(filepath.Join(configDir(), "roles", role+".md"))
	if err != nil {
		var have []string
		if entries, e := os.ReadDir(filepath.Join(configDir(), "roles")); e == nil {
			for _, en := range entries {
				if strings.HasSuffix(en.Name(), ".md") {
					have = append(have, strings.TrimSuffix(en.Name(), ".md"))
				}
			}
		}
		note := ""
		if len(have) > 0 {
			note = " (available: " + strings.Join(have, ", ") + ")"
		}
		fail("no role %q%s", role, note)
	}

	final := []string{"--append-system-prompt", string(persona)}
	if resolveSkip(skipFlag) {
		final = append([]string{"--dangerously-skip-permissions"}, final...)
	}
	final = append(final, claudeArgs...)

	if os.Getenv("ROLEPLAY_DRY") != "" {
		fmt.Printf("cwd: %s\nexec: claude %s\n", mustCwd(), shellJoin(final))
		return
	}

	cmd := exec.Command("claude", final...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fail("claude: %v (is the `claude` CLI on PATH?)", err)
	}
}

// resolveSkip: explicit flag > ROLEPLAY_SKIP env > host entry > global default.
func resolveSkip(flag *bool) bool {
	if flag != nil {
		return *flag
	}
	if v, ok := os.LookupEnv("ROLEPLAY_SKIP"); ok {
		return v == "1" || strings.EqualFold(v, "true")
	}
	c := loadConfig()
	host, _ := os.Hostname()
	for _, h := range c.SkipHosts {
		if h == host {
			return true
		}
	}
	return c.SkipPermissions
}

func listRoles() {
	dir := filepath.Join(configDir(), "roles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("(no roles yet — create one with: roleplay new <role>)")
		return
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Println("(no roles yet — create one with: roleplay new <role>)")
		return
	}
	for _, n := range names {
		fmt.Println(n)
	}
}

const scaffoldTpl = `# Role: %s

One-line identity: <what this role is for>.

## Instincts
- ...

## Domain (what you know going in)
- ...

## Tools
- ...

## Hard rules
- ...

## What you never do
- ...
`

func scaffold(role string) {
	dir := filepath.Join(configDir(), "roles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, role+".md")
	if _, err := os.Stat(path); err == nil {
		fail("%s already exists", path)
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf(scaffoldTpl, role)), 0o644); err != nil {
		fail("write %s: %v", path, err)
	}
	// Ensure a config.json exists so the user knows where per-host skip lives.
	cfgPath := filepath.Join(configDir(), "config.json")
	if _, err := os.Stat(cfgPath); err != nil {
		_ = os.WriteFile(cfgPath, []byte("{\n  \"skip_permissions\": false,\n  \"skip_hosts\": []\n}\n"), 0o644)
	}
	fmt.Printf("created %s\nedit it, then: roleplay %s\n", path, role)
}

func usage() {
	fmt.Print(`roleplay — launch claude with a role persona.

usage:
  roleplay <role> [workdir] [claude-flags...]   start a session in that role
  roleplay list                                 list roles
  roleplay new <role>                           scaffold a new role

personas live in ~/.config/roleplay/roles/<role>.md and are injected via
--append-system-prompt. memory is claude's native (CLAUDE.md / ~/.claude).

--dangerously-skip-permissions: per-host (config.json skip_hosts) + override
(--skip-permissions / --no-skip-permissions / ROLEPLAY_SKIP env).
ROLEPLAY_DRY=1 prints the command instead of running it.
`)
}

func mustCwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return d
}

// shellJoin renders args for a dry-run preview (quote only what needs it).
func shellJoin(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'") {
			parts[i] = fmt.Sprintf("%q", a)
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "roleplay: "+format+"\n", args...)
	os.Exit(1)
}
