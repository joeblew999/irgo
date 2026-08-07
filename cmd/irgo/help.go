// Help text for every command.
//
// Separated from dispatch because main.go was 916 lines, of which two thirds
// were strings: the switch that routes a command and the prose describing it
// change for different reasons and at different times.
package main

import (
	"fmt"
	"strings"
)

func printUsage() {
	fmt.Println(`irgo - one Go codebase for web, desktop, iOS and Android

Usage:
  irgo <noun> <verb> [target] [flags]

One grammar throughout: every command is a noun and a verb.

PROJECT   the repository you are in
` + renderNounSection("project") + `

APP       what gets built, run, shipped and installed
` + renderNounSection("app") + `

TOOLS     the toolchains on this machine
` + renderNounSection("tools") + `

SERVER    the development server
` + renderNounSection("server") + `

  version                  Print version information
  help [command]           Detail for one command

Examples:
  irgo project new myapp             Create a project
  irgo server dev                    Hot-reload server on :8080
  irgo tools doctor                  What can this machine build?
  irgo app run ios                   iOS Simulator
  irgo app run ios --device          A USB-connected iPhone
  irgo app build desktop all         Every desktop target this host supports
  irgo app package macos --dmg       Signed .app and a DMG
  irgo app install desktop           Put the built app in /Applications
  irgo app deploy cloudflare         Live on Cloudflare Workers, SSE included
  irgo tools remove --yes            Undo everything irgo installed
  irgo project config ios.team       Which team signs, and what is available
  irgo project pin local ../irgo     Build a checkout you are editing

Nothing needs installing first: toolchains provision themselves when a command
needs them, and go.mod pins the CLI so ` + "`go tool irgo`" + ` always matches the project.`)
}

// printCommandHelp explains one command, rendered from its declaration in
// commands.go. Nothing is typed twice: usage lines, flags and column widths
// all come from the same data the index and the README read.
func printCommandHelp(noun, verb string) {
	key := strings.TrimSpace(noun + " " + verb)

	if c, ok := commands[key]; ok {
		printCommand(key, c)
		return
	}

	switch key {
	case "project", "app", "tools", "server":
		fmt.Printf("irgo %s - %s\n\n", noun, nounSummary[noun])
		fmt.Println("Verbs:")
		fmt.Println(renderNounVerbs(noun, "irgo "))
		fmt.Printf("\nDetail for one:  irgo help %s <verb>\n", noun)

	case "version":
		fmt.Println(`irgo version - Print the version

Usage:
  irgo version

Prints the CLI version and, in a project, which irgo go.mod resolves to — a
release, a fork tag or a local checkout.`)

	default:
		fmt.Printf("No help for %q.\n\n", key)
		printUsage()
	}
}

// printCommand renders one command's page.
func printCommand(key string, c command) {
	fmt.Printf("irgo %s - %s\n", key, c.summary)

	if len(c.usage) > 0 {
		fmt.Println()
		fmt.Println("Usage:")
		width := 0
		for _, u := range c.usage {
			if n := len("irgo " + key + " " + u[0]); n > width {
				width = n
			}
		}
		for _, u := range c.usage {
			left := strings.TrimRight("irgo "+key+" "+u[0], " ")
			fmt.Printf("  %-*s  %s\n", width, left, u[1])
		}
	}

	if len(c.flags) > 0 {
		fmt.Println()
		fmt.Println("Flags:")
		width := 0
		for _, f := range c.flags {
			if len(f[0]) > width {
				width = len(f[0])
			}
		}
		for _, f := range c.flags {
			fmt.Printf("  %-*s  %s\n", width, f[0], f[1])
		}
	}

	if c.notes != "" {
		fmt.Println()
		fmt.Println(c.notes)
	}
}

// nounSummary is the one-line description of each noun, used when help is
// asked for a noun rather than a command.
var nounSummary = map[string]string{
	"project": "the repository you are in",
	"app":     "what gets built, run, shipped and installed",
	"tools":   "the toolchains on this machine",
	"server":  "the development server",
}

// renderCommandTable writes the command reference that ships in the generated
// README, from the same declarations the CLI dispatches on.
func renderCommandTable() string {
	var b strings.Builder
	b.WriteString("| Command | What it does |\n|---|---|\n")
	for _, noun := range []string{"project", "app", "tools", "server"} {
		for _, verb := range nounVerbs[noun] {
			key := noun + " " + verb
			c := commands[key]
			name := key
			if len(c.targets) > 0 {
				name += " <" + strings.Join(c.targets, "|") + ">"
			}
			fmt.Fprintf(&b, "| `%s` | %s |\n", name, c.summary)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderNounSection writes a noun's verbs for the usage index, with the
// targets or arguments each accepts and the columns lined up.
func renderNounSection(noun string) string { return renderNounVerbs(noun, "") }

// renderNounVerbs is the same listing with an optional prefix: the index
// writes bare command names, a noun's own help page writes runnable ones.
func renderNounVerbs(noun, prefix string) string {
	type row struct{ left, right string }
	var rows []row
	width := 0
	for _, verb := range nounVerbs[noun] {
		c := commands[noun+" "+verb]
		left := prefix + noun + " " + verb
		if len(c.targets) > 0 {
			left += " <" + strings.Join(c.targets, "|") + ">"
		} else if c.args != "" {
			left += " " + c.args
		}
		if len(left) > width {
			width = len(left)
		}
		rows = append(rows, row{left, c.summary})
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, r.left, r.right)
	}
	return strings.TrimRight(b.String(), "\n")
}
