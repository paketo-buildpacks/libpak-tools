/*
 * Copyright 2018-2026 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ci

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
)

const (
	// SemverBumpNone means no dependency version changed.
	SemverBumpNone = "none"
	// SemverBumpPatch means the highest dependency change was a patch bump.
	SemverBumpPatch = "patch"
	// SemverBumpMinor means the highest dependency change was a minor bump.
	SemverBumpMinor = "minor"
	// SemverBumpMajor means the highest dependency change was a major bump.
	SemverBumpMajor = "major"
)

var bumpRank = map[string]int{
	SemverBumpNone:  0,
	SemverBumpPatch: 1,
	SemverBumpMinor: 2,
	SemverBumpMajor: 3,
}

type buildpackConfig struct {
	Metadata struct {
		Dependencies []buildpackDependency `toml:"dependencies"`
	} `toml:"metadata"`
}

type buildpackDependency struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
}

type dependencyKey struct {
	ID    string
	Major uint64
}

type dependencyVersions map[string]*semver.Version

type dependencyGroups map[dependencyKey]dependencyVersions

// CalculateSemverBump returns the highest semver bump between two
// buildpack.toml files. Dependencies are grouped by ID and major version so
// updates to older lines, such as an older LTS runtime, are not hidden by a
// newer line using a larger major version.
func CalculateSemverBump(beforePath, afterPath string) (string, error) {
	before, err := readDependencyGroups(beforePath)
	if err != nil {
		return "", err
	}

	after, err := readDependencyGroups(afterPath)
	if err != nil {
		return "", err
	}

	return calculateBump(before, after), nil
}

func readDependencyGroups(path string) (dependencyGroups, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s\n%w", path, err)
	}

	config := buildpackConfig{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unable to parse %s\n%w", path, err)
	}

	groups := dependencyGroups{}
	for _, dependency := range config.Metadata.Dependencies {
		if dependency.ID == "" || dependency.Version == "" {
			continue
		}

		version, err := semver.NewVersion(dependency.Version)
		if err != nil {
			continue
		}

		key := dependencyKey{ID: dependency.ID, Major: version.Major()}
		if _, ok := groups[key]; !ok {
			groups[key] = dependencyVersions{}
		}

		groups[key][dependency.Version] = version
	}

	return groups, nil
}

func calculateBump(before, after dependencyGroups) string {
	highest := SemverBumpNone

	for key, afterVersions := range after {
		beforeVersions, ok := before[key]
		if !ok {
			// A dependency line that did not exist before is a new major
			// version line.
			highest = maxBump(highest, SemverBumpMajor)
			continue
		}

		if sameVersions(beforeVersions, afterVersions) {
			continue
		}

		bump := compareVersions(maxVersion(beforeVersions), maxVersion(afterVersions))
		if bump == SemverBumpNone {
			// The version strings changed, but the semver precedence did not,
			// for example when only build metadata changed. Treat the
			// dependency update as a patch.
			bump = SemverBumpPatch
		}

		highest = maxBump(highest, bump)
	}

	return highest
}

func sameVersions(a, b dependencyVersions) bool {
	if len(a) != len(b) {
		return false
	}

	for version := range a {
		if _, ok := b[version]; !ok {
			return false
		}
	}

	return true
}

func maxVersion(versions dependencyVersions) *semver.Version {
	var maximum *semver.Version
	for _, version := range versions {
		if maximum == nil || version.GreaterThan(maximum) {
			maximum = version
		}
	}

	return maximum
}

func compareVersions(before, after *semver.Version) string {
	if before == nil || after == nil || after.Compare(before) <= 0 {
		return SemverBumpNone
	}

	if after.Major() > before.Major() {
		return SemverBumpMajor
	}

	if after.Minor() > before.Minor() {
		return SemverBumpMinor
	}

	return SemverBumpPatch
}

func maxBump(a, b string) string {
	if bumpRank[b] > bumpRank[a] {
		return b
	}

	return a
}
