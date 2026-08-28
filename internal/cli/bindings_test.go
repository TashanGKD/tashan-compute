package cli

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestCommandTreeCoversEveryCapabilityBinding(t *testing.T) {
	contents, err := os.ReadFile("bindings.json")
	if err != nil {
		t.Fatalf("ReadFile(bindings.json) error = %v", err)
	}
	var bindings map[string]string
	if err := json.Unmarshal(contents, &bindings); err != nil {
		t.Fatalf("Unmarshal(bindings.json) error = %v", err)
	}
	commands := make(map[string]struct{})
	root := NewRoot(Dependencies{})
	var visit func(prefix string, commandNames []string)
	visit = func(prefix string, commandNames []string) {
		for _, name := range commandNames {
			path := strings.TrimSpace(prefix + " " + name)
			commands[path] = struct{}{}
			command, _, err := root.Find(strings.Fields(path))
			if err != nil {
				continue
			}
			children := command.Commands()
			names := make([]string, 0, len(children))
			for _, child := range children {
				if child.Name() != "help" && child.Name() != "completion" {
					names = append(names, child.Name())
				}
			}
			visit(path, names)
		}
	}
	top := root.Commands()
	var topNames []string
	for _, command := range top {
		if command.Name() != "help" && command.Name() != "completion" {
			topNames = append(topNames, command.Name())
		}
	}
	visit("", topNames)

	var missing []string
	for capabilityID, command := range bindings {
		if _, exists := commands[command]; !exists {
			missing = append(missing, capabilityID+" -> "+command)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("missing real CLI commands:\n%s", strings.Join(missing, "\n"))
	}
}
