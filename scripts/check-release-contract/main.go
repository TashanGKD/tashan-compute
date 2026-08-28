package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
)

type releaseMetadata struct {
	SchemaVersion int               `json:"schemaVersion"`
	Version       string            `json:"version"`
	Repository    string            `json:"repository"`
	Platforms     []releasePlatform `json:"platforms"`
}

type releasePlatform struct {
	ID    string `json:"id"`
	Asset string `json:"asset"`
}

var semver = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z]+)*)?$`)

func main() {
	root := flag.String("root", ".", "repository or fixture root")
	flag.Parse()
	if err := checkRelease(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkRelease(root string) error {
	primary, err := readRelease(filepath.Join(root, "release", "cli-release.json"))
	if err != nil {
		return err
	}
	skill, err := readRelease(filepath.Join(root, "skill", "tashan-compute", "release.json"))
	if err != nil {
		return err
	}
	if err := validateRelease(primary); err != nil {
		return err
	}
	if err := validateRelease(skill); err != nil {
		return err
	}
	if !reflect.DeepEqual(primary, skill) {
		return errors.New("release metadata drift")
	}
	fmt.Printf("release contract: PASS (%s)\n", primary.Version)
	return nil
}

func readRelease(path string) (releaseMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return releaseMetadata{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var metadata releaseMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return releaseMetadata{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return releaseMetadata{}, errors.New("release metadata has trailing JSON")
	}
	return metadata, nil
}

func validateRelease(metadata releaseMetadata) error {
	if metadata.SchemaVersion != 1 || !semver.MatchString(metadata.Version) || metadata.Repository == "" {
		return errors.New("invalid release metadata")
	}
	wanted := map[string]bool{"darwin-arm64": false, "darwin-x64": false, "linux-x64": false}
	for _, platform := range metadata.Platforms {
		if _, ok := wanted[platform.ID]; !ok || wanted[platform.ID] {
			return fmt.Errorf("invalid or duplicate release platform: %s", platform.ID)
		}
		wanted[platform.ID] = true
		expected := fmt.Sprintf("tcompute-v%s-%s.tar.gz", metadata.Version, platform.ID)
		if platform.Asset != expected {
			return fmt.Errorf("asset name mismatch: %s", platform.ID)
		}
	}
	for platform, found := range wanted {
		if !found {
			return fmt.Errorf("missing release platform: %s", platform)
		}
	}
	return nil
}
