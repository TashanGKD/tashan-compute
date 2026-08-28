package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TashanGKD/tashan-compute/internal/capability"
)

func main() {
	root := flag.String("root", ".", "repository or fixture root")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	manifestFile, err := os.Open(filepath.Join(root, "capabilities", "manifest.json"))
	if err != nil {
		return fmt.Errorf("open capability manifest: %w", err)
	}
	defer manifestFile.Close()

	manifest, err := capability.Load(manifestFile)
	if err != nil {
		return err
	}

	bindings := make(map[string]string)
	if err := decodeFile(filepath.Join(root, "internal", "cli", "bindings.json"), &bindings); err != nil {
		return fmt.Errorf("decode CLI bindings: %w", err)
	}
	if err := capability.CheckBindings(manifest, bindings); err != nil {
		return err
	}

	var references []string
	if err := decodeFile(filepath.Join(root, "skill", "tashan-compute", "capability-references.json"), &references); err != nil {
		return fmt.Errorf("decode Skill references: %w", err)
	}
	known := make(map[string]struct{}, len(manifest))
	for _, item := range manifest {
		known[item.ID] = struct{}{}
	}
	referenced := make(map[string]struct{}, len(references))
	for _, id := range references {
		if _, ok := known[id]; !ok {
			return fmt.Errorf("unknown Skill capability: %s", id)
		}
		if _, duplicate := referenced[id]; duplicate {
			return fmt.Errorf("duplicate Skill capability: %s", id)
		}
		referenced[id] = struct{}{}
	}
	for id := range known {
		if _, ok := referenced[id]; !ok {
			return fmt.Errorf("missing Skill capability: %s", id)
		}
	}

	fmt.Printf("capability coverage: PASS (%d capabilities)\n", len(manifest))
	return nil
}

func decodeFile(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}
