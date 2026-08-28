package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func main() {
	root := flag.String("root", ".", "repository or fixture root")
	flag.Parse()
	if err := check(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func check(root string) error {
	skillRoot := filepath.Join(root, "skill", "tashan-compute")
	contents, err := os.ReadFile(filepath.Join(skillRoot, "SKILL.md"))
	if err != nil {
		return err
	}
	text := string(contents)
	if !strings.HasPrefix(text, "---\n") {
		return errors.New("Skill frontmatter is missing")
	}
	name := frontmatterValue(text, "name")
	description := frontmatterValue(text, "description")
	if !skillNamePattern.MatchString(name) || name != "tashan-compute" {
		return errors.New("invalid Skill name")
	}
	if !strings.HasPrefix(description, "Use when ") {
		return errors.New("Skill description must start with Use when")
	}
	for _, relative := range []string{"agents/openai.yaml", "release.json", "scripts/install-cli.sh", "references/authentication.md", "references/security.md", "capability-references.json"} {
		info, err := os.Stat(filepath.Join(skillRoot, relative))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("required Skill file missing: %s", relative)
		}
	}
	installer, _ := os.Stat(filepath.Join(skillRoot, "scripts", "install-cli.sh"))
	if installer.Mode().Perm()&0o111 == 0 {
		return errors.New("Skill installer is not executable")
	}
	fmt.Println("skill structure: PASS")
	return nil
}

func frontmatterValue(contents, key string) string {
	for index, line := range strings.Split(contents, "\n") {
		if index == 0 {
			continue
		}
		if line == "---" {
			break
		}
		if strings.HasPrefix(line, key+": ") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		}
	}
	return ""
}
