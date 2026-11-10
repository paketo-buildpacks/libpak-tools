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

package ci_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak-tools/ci"
)

type dependency struct {
	id      string
	version string
}

func testSemver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		dir    string
		before string
		after  string
	)

	it.Before(func() {
		dir = t.TempDir()
		before = filepath.Join(dir, "before.toml")
		after = filepath.Join(dir, "after.toml")
	})

	context("version bumps", func() {
		testCases := []struct {
			name     string
			before   []dependency
			after    []dependency
			expected string
		}{
			{
				name:     "no version changes",
				before:   []dependency{{"jdk", "11.0.31"}},
				after:    []dependency{{"jdk", "11.0.31"}},
				expected: ci.SemverBumpNone,
			},
			{
				name:     "patch version changes",
				before:   []dependency{{"jdk", "11.0.31"}},
				after:    []dependency{{"jdk", "11.0.32"}},
				expected: ci.SemverBumpPatch,
			},
			{
				name:     "minor version changes",
				before:   []dependency{{"jdk", "11.0.31"}},
				after:    []dependency{{"jdk", "11.1.0"}},
				expected: ci.SemverBumpMinor,
			},
			{
				name:     "major version changes",
				before:   []dependency{{"jdk", "11.0.31"}},
				after:    []dependency{{"jdk", "12.0.0"}},
				expected: ci.SemverBumpMajor,
			},
			{
				name: "multiple changes return the highest bump",
				before: []dependency{
					{"jdk", "1.0.0"},
					{"jre", "1.0.0"},
					{"node", "1.0.0"},
				},
				after: []dependency{
					{"jdk", "1.0.1"},
					{"jre", "1.1.0"},
					{"node", "2.0.0"},
				},
				expected: ci.SemverBumpMajor,
			},
			{
				name: "does not hide an update to an older major line",
				before: []dependency{
					{"jdk", "11.0.31"},
					{"jdk", "21.0.11"},
				},
				after: []dependency{
					{"jdk", "11.0.32"},
					{"jdk", "21.0.11"},
				},
				expected: ci.SemverBumpPatch,
			},
			{
				name: "treats a new major line as a major bump",
				before: []dependency{
					{"jdk", "11.0.31"},
					{"jdk", "21.0.11"},
				},
				after: []dependency{
					{"jdk", "11.0.31"},
					{"jdk", "21.0.11"},
					{"jdk", "25.0.4"},
				},
				expected: ci.SemverBumpMajor,
			},
			{
				name: "returns a single bump for duplicate versions",
				before: []dependency{
					{"jdk", "11.0.31"},
					{"jdk", "11.0.31"},
				},
				after: []dependency{
					{"jdk", "11.0.32"},
					{"jdk", "11.0.32"},
				},
				expected: ci.SemverBumpPatch,
			},
			{
				name:     "returns patch for build metadata changes",
				before:   []dependency{{"jdk", "11.0.32+101"}},
				after:    []dependency{{"jdk", "11.0.32+102"}},
				expected: ci.SemverBumpPatch,
			},
			{
				name:     "returns patch for pre-release changes",
				before:   []dependency{{"jdk", "1.0.0-rc1"}},
				after:    []dependency{{"jdk", "1.0.0-rc2"}},
				expected: ci.SemverBumpPatch,
			},
			{
				name: "ignores unparseable versions",
				before: []dependency{
					{"jdk", "11.0.31"},
					{"jre", "not-a-version"},
				},
				after: []dependency{
					{"jdk", "11.0.32"},
					{"jre", "also-not-a-version"},
				},
				expected: ci.SemverBumpPatch,
			},
		}

		for _, testCase := range testCases {
			it(testCase.name, func() {
				writeBuildpackTOML(t, before, testCase.before...)
				writeBuildpackTOML(t, after, testCase.after...)

				bump, err := ci.CalculateSemverBump(before, after)
				Expect(err).NotTo(HaveOccurred())
				Expect(bump).To(Equal(testCase.expected))
			})
		}
	})

	context("invalid input", func() {
		it("fails when the before file is missing", func() {
			writeBuildpackTOML(t, after, dependency{"jdk", "11.0.32"})

			_, err := ci.CalculateSemverBump(filepath.Join(dir, "missing.toml"), after)
			Expect(err).To(MatchError(ContainSubstring("unable to read")))
		})

		it("fails when a file is not valid TOML", func() {
			Expect(os.WriteFile(before, []byte("[metadata"), 0600)).To(Succeed())
			writeBuildpackTOML(t, after, dependency{"jdk", "11.0.32"})

			_, err := ci.CalculateSemverBump(before, after)
			Expect(err).To(MatchError(ContainSubstring("unable to parse")))
		})
	})
}

func writeBuildpackTOML(t *testing.T, path string, dependencies ...dependency) {
	t.Helper()

	var contents strings.Builder
	contents.WriteString("[metadata]\n")

	for _, dependency := range dependencies {
		fmt.Fprintln(&contents, "\n  [[metadata.dependencies]]")
		fmt.Fprintf(&contents, "    id = %q\n", dependency.id)
		fmt.Fprintf(&contents, "    version = %q\n", dependency.version)
	}

	if err := os.WriteFile(path, []byte(contents.String()), 0600); err != nil {
		t.Fatalf("unable to write %s: %s", path, err)
	}
}
