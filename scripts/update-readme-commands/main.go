package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type commandEntry struct {
	parts []string
	full  string
}

func main() {
	const readmePath = "README.md"

	content, err := os.ReadFile(readmePath)
	if err != nil {
		failf("unable to read %s: %v", readmePath, err)
	}

	text := string(content)
	bodyStart, bodyEnd, err := findSectionBodyBounds(text, "Commands")
	if err != nil {
		failf("%v", err)
	}
	body := text[bodyStart:bodyEnd]

	manualDescriptions := parseManualDescriptions(body)
	commands, err := discoverCommands()
	if err != nil {
		failf("unable to discover commands: %v", err)
	}

	newBody, err := renderCommandsSection(commands, manualDescriptions)
	if err != nil {
		failf("unable to render commands section: %v", err)
	}

	updated := text[:bodyStart] + newBody + text[bodyEnd:]
	if err := os.WriteFile(readmePath, []byte(updated), 0600); err != nil {
		failf("unable to write %s: %v", readmePath, err)
	}
}

func parseManualDescriptions(body string) map[string]string {
	desc := map[string]string{}

	headingRE := regexp.MustCompile(`(?m)^###\s+` + "`" + `([^` + "`" + `]+)` + "`" + `\s*$`)
	entries := headingRE.FindAllStringSubmatchIndex(body, -1)
	for i, entry := range entries {
		cmd := strings.TrimSpace(body[entry[2]:entry[3]])
		blockStart := entry[1]
		blockEnd := len(body)
		if i+1 < len(entries) {
			blockEnd = entries[i+1][0]
		}
		block := body[blockStart:blockEnd]
		parts := strings.SplitN(block, "```", 2)
		manual := strings.Trim(parts[0], "\n")
		if strings.TrimSpace(manual) != "" {
			desc[cmd] = manual
		}
	}

	return desc
}

func findSectionBodyBounds(text, sectionTitle string) (int, int, error) {
	headerRE := regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(sectionTitle) + `\s*$`)
	headerIdx := headerRE.FindStringIndex(text)
	if headerIdx == nil {
		return 0, 0, fmt.Errorf("README.md must contain a '## %s' section", sectionTitle)
	}

	bodyStart := headerIdx[1]
	if bodyStart < len(text) && text[bodyStart] == '\n' {
		bodyStart++
	}

	nextHeaderRE := regexp.MustCompile(`(?m)^##\s+`)
	next := nextHeaderRE.FindStringIndex(text[bodyStart:])
	if next == nil {
		return bodyStart, len(text), nil
	}

	return bodyStart, bodyStart + next[0], nil
}

func discoverCommands() ([]commandEntry, error) {
	seen := map[string]struct{}{}
	queue := [][]string{{}}
	entries := []commandEntry{}

	for len(queue) > 0 {
		parts := queue[0]
		queue = queue[1:]
		if len(parts) > 0 && parts[0] == "completion" {
			continue
		}

		full := "libpak-tools"
		if len(parts) > 0 {
			full += " " + strings.Join(parts, " ")
		}

		if _, ok := seen[full]; ok {
			continue
		}
		seen[full] = struct{}{}
		entries = append(entries, commandEntry{parts: parts, full: full})

		help, err := runHelp(parts)
		if err != nil {
			return nil, err
		}

		for _, sub := range parseSubcommands(help) {
			next := append([]string{}, parts...)
			next = append(next, sub)
			queue = append(queue, next)
		}
	}

	return entries, nil
}

func runHelp(parts []string) (string, error) {
	args := append([]string{"run", "."}, parts...)
	args = append(args, "--help")

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go %s failed: %w\n%s", strings.Join(args, " "), err, string(output))
	}

	return strings.TrimRight(string(output), "\n"), nil
}

func parseSubcommands(help string) []string {
	subs := []string{}
	inAvailable := false

	for _, line := range strings.Split(help, "\n") {
		if strings.TrimSpace(line) == "Available Commands:" {
			inAvailable = true
			continue
		}

		if !inAvailable {
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		if !strings.HasPrefix(line, "  ") {
			break
		}

		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}

		name := fields[0]
		if name == "help" || name == "completion" {
			continue
		}

		subs = append(subs, name)
	}

	return subs
}

func renderCommandsSection(commands []commandEntry, manual map[string]string) (string, error) {
	sections := []string{}

	for _, c := range commands {
		help, err := runHelp(c.parts)
		if err != nil {
			return "", err
		}

		var b strings.Builder
		b.WriteString("### `")
		b.WriteString(c.full)
		b.WriteString("`\n\n")

		if d, ok := manual[c.full]; ok {
			b.WriteString(strings.TrimRight(d, "\n"))
			b.WriteString("\n\n")
		}

		b.WriteString("```\n")
		b.WriteString("> ")
		b.WriteString(helpInvocation(c.parts))
		b.WriteString("\n")
		b.WriteString(help)
		b.WriteString("\n```")

		sections = append(sections, b.String())
	}

	return strings.Join(sections, "\n\n") + "\n\n", nil
}

func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func helpInvocation(parts []string) string {
	cmd := "libpak-tools"
	if len(parts) > 0 {
		cmd += " " + strings.Join(parts, " ")
	}
	return cmd + " --help"
}
