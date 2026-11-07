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

	context("no version changes", func() {
		it("returns none", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "11.0.31"})
			writeBuildpackTOML(t, after, dependency{"jdk", "11.0.31"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpNone))
		})
	})

	context("patch version changes", func() {
		it("returns patch", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "11.0.31"})
			writeBuildpackTOML(t, after, dependency{"jdk", "11.0.32"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})
	})

	context("minor version changes", func() {
		it("returns minor", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "11.0.31"})
			writeBuildpackTOML(t, after, dependency{"jdk", "11.1.0"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpMinor))
		})
	})

	context("major version changes", func() {
		it("returns major", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "11.0.31"})
			writeBuildpackTOML(t, after, dependency{"jdk", "12.0.0"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpMajor))
		})
	})

	context("multiple changes", func() {
		it("returns the highest bump", func() {
			writeBuildpackTOML(t, before,
				dependency{"jdk", "1.0.0"},
				dependency{"jre", "1.0.0"},
				dependency{"node", "1.0.0"},
			)
			writeBuildpackTOML(t, after,
				dependency{"jdk", "1.0.1"},
				dependency{"jre", "1.1.0"},
				dependency{"node", "2.0.0"},
			)

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpMajor))
		})
	})

	context("multiple major version lines", func() {
		it("does not hide an update to an older line", func() {
			writeBuildpackTOML(t, before,
				dependency{"jdk", "11.0.31"},
				dependency{"jdk", "21.0.11"},
			)
			writeBuildpackTOML(t, after,
				dependency{"jdk", "11.0.32"},
				dependency{"jdk", "21.0.11"},
			)

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})

		it("treats a new major line as a major bump", func() {
			writeBuildpackTOML(t, before,
				dependency{"jdk", "11.0.31"},
				dependency{"jdk", "21.0.11"},
			)
			writeBuildpackTOML(t, after,
				dependency{"jdk", "11.0.31"},
				dependency{"jdk", "21.0.11"},
				dependency{"jdk", "25.0.4"},
			)

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpMajor))
		})
	})

	context("duplicate versions for multiple targets", func() {
		it("returns a single bump", func() {
			writeBuildpackTOML(t, before,
				dependency{"jdk", "11.0.31"},
				dependency{"jdk", "11.0.31"},
			)
			writeBuildpackTOML(t, after,
				dependency{"jdk", "11.0.32"},
				dependency{"jdk", "11.0.32"},
			)

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})
	})

	context("build metadata changes", func() {
		it("returns patch", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "11.0.32+101"})
			writeBuildpackTOML(t, after, dependency{"jdk", "11.0.32+102"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})
	})

	context("pre-release changes", func() {
		it("returns patch", func() {
			writeBuildpackTOML(t, before, dependency{"jdk", "1.0.0-rc1"})
			writeBuildpackTOML(t, after, dependency{"jdk", "1.0.0-rc2"})

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})
	})

	context("unparseable versions", func() {
		it("ignores them", func() {
			writeBuildpackTOML(t, before,
				dependency{"jdk", "11.0.31"},
				dependency{"jre", "not-a-version"},
			)
			writeBuildpackTOML(t, after,
				dependency{"jdk", "11.0.32"},
				dependency{"jre", "also-not-a-version"},
			)

			bump, err := ci.CalculateSemverBump(before, after)
			Expect(err).NotTo(HaveOccurred())
			Expect(bump).To(Equal(ci.SemverBumpPatch))
		})
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
