package console

import (
	"fmt"
	"os"
	"strings"
)

// Command is a console command: name, description and run function. Args are
// the raw command-line arguments after the command name.
type Command struct {
	Name        string
	Description string
	Run         func(args []string) int
}

// commands returns the registry of built-in commands bound to this kernel.
func (k *Kernel) commands() []Command {
	return []Command{
		{
			Name:        "route:list",
			Description: "List all registered routes",
			Run:         func(args []string) int { return runRouteList(k.dumpRoutes, args) },
		},
		{
			Name:        "make:controller",
			Description: "Create a new controller file (make:controller Name [--api])",
			Run:         runMakeController,
		},
		{
			Name:        "make:middleware",
			Description: "Create a new middleware file (make:middleware Name)",
			Run:         runMakeMiddleware,
		},
	}
}

// help prints the available commands.
func (k *Kernel) help() {
	fmt.Println("Available commands:")
	for _, c := range k.commands() {
		fmt.Printf("  %-20s %s\n", c.Name, c.Description)
	}
}

// routeDump is injected by the think package when the kernel is bound.
var routeDump func() (string, bool)

// SetRouteDump injects the route-table accessor.
func SetRouteDump(fn func() (string, bool)) {
	routeDump = fn
}

// runRouteList prints the route table with an optional filter:
// route:list [--path=/prefix] [--name=substring] [--json].
func runRouteList(dump func() (string, bool), args []string) int {
	routeDump, ok := dump()
	if !ok {
		fmt.Fprintln(os.Stderr, "router is not available")
		return 1
	}

	pathFilter, nameFilter, asJSON := "", "", false
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--path="):
			pathFilter = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "--name="):
			nameFilter = strings.TrimPrefix(arg, "--name=")
		case arg == "--json":
			asJSON = true
		}
	}

	lines := strings.Split(strings.TrimRight(routeDump, "\n"), "\n")
	if asJSON {
		fmt.Println("[")
		first := true
		for _, line := range lines {
			method, uri, rest := splitDumpLine(line)
			if method == "" {
				continue
			}
			if pathFilter != "" && !strings.Contains(uri, pathFilter) {
				continue
			}
			if nameFilter != "" && !strings.Contains(rest, nameFilter) {
				continue
			}
			if !first {
				fmt.Println(",")
			}
			first = false
			fmt.Printf("  {\"method\": %q, \"uri\": %q, \"action\": %q}", method, uri, rest)
		}
		fmt.Println("\n]")
		return 0
	}

	for _, line := range lines {
		method, uri, rest := splitDumpLine(line)
		if method == "" {
			continue
		}
		if pathFilter != "" && !strings.Contains(uri, pathFilter) {
			continue
		}
		if nameFilter != "" && !strings.Contains(rest, nameFilter) {
			continue
		}
		fmt.Println(line)
	}
	return 0
}

// splitDumpLine splits a dump line "METHODS URI ACTION" into parts.
func splitDumpLine(line string) (method, uri, rest string) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return "", "", ""
	}
	method, uri = fields[0], fields[1]
	rest = strings.Join(fields[2:], " ")
	return method, uri, rest
}

// runMakeController generates a controller file.
func runMakeController(args []string) int {
	name, api := "", false
	for _, arg := range args {
		switch {
		case arg == "--api":
			api = true
		case name == "":
			name = arg
		}
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: make:controller Name [--api]")
		return 1
	}

	actions := []string{"Index", "Store", "Show", "Update", "Destroy"}
	if !api {
		actions = append([]string{"Create"}, append(actions[:2:2], append([]string{"Edit"}, actions[2:]...)...)...)
	}

	var b strings.Builder
	b.WriteString("package controllers\n\n")
	b.WriteString("// " + name + " handles the " + name + " resource routes.\n")
	b.WriteString("type " + name + " struct{}\n\n")
	for _, a := range actions {
		b.WriteString("// " + a + " handles the " + strings.ToLower(a) + " action.\n")
		b.WriteString("func (c *" + name + ") " + a + "(req *flow.Request) *flow.Response {\n")
		b.WriteString("\treturn flow.Text(\"" + name + "@" + a + "\")\n}\n\n")
	}
	return writeFile("app/http/controllers/"+name+".go", b.String())
}

// runMakeMiddleware generates a middleware file.
func runMakeMiddleware(args []string) int {
	name := ""
	for _, arg := range args {
		if name == "" {
			name = arg
		}
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: make:middleware Name")
		return 1
	}

	var b strings.Builder
	b.WriteString("package middleware\n\n")
	b.WriteString("import \"github.com/go-think/flow\"\n\n")
	b.WriteString("// " + name + " middleware.\n")
	b.WriteString("type " + name + " struct{}\n\n")
	b.WriteString("// New" + name + " creates the middleware.\n")
	b.WriteString("func New" + name + "() flow.Handler {\n")
	b.WriteString("\treturn &struct{}{}\n}\n\n")
	b.WriteString("// Process handles the request.\n")
	b.WriteString("func (m *" + name + ") Process(req *flow.Request, next flow.Closure) any {\n")
	b.WriteString("\treturn next(req)\n}\n")
	return writeFile("app/http/middleware/"+name+".go", b.String())
}

// writeFile creates the file (and its directory) unless it already exists.
func writeFile(path, content string) int {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", path)
		return 1
	}
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		if err := os.MkdirAll(path[:idx], 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s created successfully.\n", path)
	return 0
}
