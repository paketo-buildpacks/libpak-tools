/*
 * Copyright 2018-2025 the original author or authors.
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
package builder_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak-tools/builder"
)

func testBuilder(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("updates buildpack.toml data", func() {
		it("updates keys in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"buildpack": map[string]interface{}{
					"id":          "id",
					"name":        "name",
					"description": "desc",
					"homepage":    "homepage",
					"version":     "version",
					"foo":         "bar",
				},
			}

			builder.UpdateBuildpackDetails(builder.JVMVendor{"paketo-buildpacks/new-id", false, "new-desc", "new-homepage", "new-name", "new-id"}, "new-version")(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("buildpack"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("description", "new-desc"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("homepage", "new-homepage"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("id", "paketo-buildpacks/new-id"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("name", "new-name"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("version", "new-version"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("foo", "bar"))
		})

		it("adds keys that dont exist in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"buildpack": map[string]interface{}{
					"id":   "id",
					"name": "name",
					"foo":  "bar",
				},
			}

			builder.UpdateBuildpackDetails(builder.JVMVendor{"paketo-buildpacks/new-id", false, "new-desc", "new-homepage", "new-name", "new-id"}, "new-version")(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("buildpack"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("description", "new-desc"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("homepage", "new-homepage"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("id", "paketo-buildpacks/new-id"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("name", "new-name"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("version", "new-version"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("foo", "bar"))
		})
	})

	context("updates buildpack.toml configurations", func() {
		it("updates keys in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"metadata": map[string]interface{}{
					"configurations": []map[string]interface{}{
						{
							"name":    "foo",
							"default": "test-default",
						},
						{
							"name":    "baz",
							"default": "other-default",
						},
					},
				},
			}

			builder.UpdateBuildpackConfiguration(map[string]interface{}{
				"foo": "update-value",
			})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("configurations"))

			cfgList := buildpackTOML["metadata"].(map[string]interface{})["configurations"].([]map[string]interface{})
			Expect(cfgList).To(HaveLen(2))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("name", "foo")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("default", "update-value")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("name", "baz")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("default", "other-default")))
		})
	})

	context("metadata has a list of dependencies", func() {
		var buildpackTOML map[string]interface{}

		it.Before(func() {
			buildpackTOML = map[string]interface{}{
				"metadata": map[string]interface{}{
					"dependencies": []map[string]interface{}{
						{
							"id": "jdk-foo",
						},
						{
							"id": "jre-foo",
						},
						{
							"id": "jre-bar",
						},
						{
							"id": "jdk-baz",
						},
					},
				},
			}
		})

		it("removes everything", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(0))
		})

		it("removes dependencies not in the vendor list", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{"foo"})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(2))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(
				ContainElements(HaveKeyWithValue("id", "jdk-foo"), HaveKeyWithValue("id", "jre-foo")))
		})

		it("removes dependencies not in the vendor list", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{"baz"})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(1))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(
				ContainElements(HaveKeyWithValue("id", "jdk-baz")))
		})
	})

	context("build JVM vendor command behavior", func() {
		var baseBuildpackTOML = `[buildpack]
id = "paketo-buildpacks/original"
name = "Original"
description = "Original description"
homepage = "https://original.example"
version = "0.0.1"

[[metadata.configurations]]
name = "BP_JVM_VENDORS"
default = "temurin"

[[metadata.configurations]]
name = "BP_JVM_VENDOR"
default = "temurin"

[[metadata.dependencies]]
id = "jdk-temurin"
`

		it("infers buildpack path from BP_ROOT and buildpack id", func() {
			t.Setenv("BP_ROOT", "/workspace")

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs: []string{"paketo-community/java@1.2.3"},
			}

			Expect(cmd.InferBuildpackPath()).To(Succeed())
			Expect(cmd.BuildpackPath).To(Equal(filepath.Join("/workspace", "paketo-community", "java")))
		})

		it("errors inferring buildpack path when buildpack id format is invalid", func() {
			t.Setenv("BP_ROOT", "/workspace")

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs: []string{"paketo-community/java"},
			}

			Expect(cmd.InferBuildpackPath()).To(MatchError("invalid buildpack id: paketo-community/java, must match format 'org/name@version'"))
		})

		it("fails execute when a backup buildpack.toml already exists", func() {
			tmpDir := t.TempDir()
			buildpackPath := filepath.Join(tmpDir, "buildpack.toml")
			backupPath := filepath.Join(tmpDir, "buildpack.toml.bak")
			Expect(os.WriteFile(buildpackPath, []byte(""), 0600)).To(Succeed())
			Expect(os.WriteFile(backupPath, []byte("already-here"), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackTOMLPath:       buildpackPath,
				BuildpackTOMLBackupPath: backupPath,
			}

			Expect(cmd.Execute()).To(MatchError(ContainSubstring("backup copy of original buildpack.toml already exists")))
		})

		it("fails single-buildpack mode without buildpack ids", func() {
			cmd := builder.BuildJvmVendorsCommand{}
			Expect(cmd.BuildSingleBuildpack()).To(MatchError("no buildpack IDs specified, you must specify one buildpack ID if using --single-buildpack"))
		})

		it("validates single-buildpack id format before build", func() {
			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs: []string{"paketo-buildpacks/java"},
			}
			Expect(cmd.BuildSingleBuildpack()).To(MatchError("invalid buildpack ID: paketo-buildpacks/java, must contain two parts that are `@` separated"))
		})

		it("fails single-buildpack mode when no default vendor can be selected", func() {
			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs: []string{"paketo-buildpacks/java@1.2.3"},
			}
			Expect(cmd.BuildSingleBuildpack()).To(MatchError(ContainSubstring("unable to select default vendor")))
		})

		it("builds a single buildpack and updates toml before packaging", func() {
			t.Setenv("PATH", "")
			tmpDir := t.TempDir()
			buildpackTOMLPath := filepath.Join(tmpDir, "buildpack.toml")
			Expect(os.WriteFile(buildpackTOMLPath, []byte(baseBuildpackTOML), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs:      []string{"paketo-buildpacks/java@1.2.3"},
				SelectedVendors:   []string{"temurin"},
				DefaultVendor:     "temurin",
				BuildpackPath:     tmpDir,
				BuildpackTOMLPath: buildpackTOMLPath,
				JVMVendors: []builder.JVMVendor{
					{
						BuildpackID: "paketo-buildpacks/temurin",
						Description: "Temurin buildpack",
						Homepage:    "https://adoptium.net",
						Name:        "Temurin",
						VendorID:    "temurin",
					},
				},
			}

			Expect(cmd.BuildSingleBuildpack()).To(HaveOccurred())
			contents, err := os.ReadFile(buildpackTOMLPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("default = \"temurin\""))
		})

		it("fails multiple-buildpack mode when selected vendors and buildpack ids are mismatched", func() {
			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs:    []string{"paketo-buildpacks/java@1.2.3"},
				SelectedVendors: []string{"liberica", "temurin"},
			}
			Expect(cmd.BuildMultipleBuildpacks()).To(MatchError(ContainSubstring("must match number of selected vendors")))
		})

		it("fails multiple-buildpack mode when buildpack.toml backup is missing", func() {
			tmpDir := t.TempDir()
			buildpackPath := filepath.Join(tmpDir, "buildpack.toml")
			Expect(os.WriteFile(buildpackPath, []byte(""), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs:            []string{"paketo-buildpacks/java@1.2.3"},
				SelectedVendors:         []string{"temurin"},
				BuildpackTOMLPath:       buildpackPath,
				BuildpackTOMLBackupPath: filepath.Join(tmpDir, "buildpack.toml.bak"),
				JVMVendors: []builder.JVMVendor{
					{VendorID: "temurin"},
				},
			}

			Expect(cmd.BuildMultipleBuildpacks()).To(MatchError(ContainSubstring("failed to customize buildpack.toml")))
		})

		it("attempts multiple buildpack packaging after customization", func() {
			t.Setenv("PATH", "")
			tmpDir := t.TempDir()
			buildpackTOMLPath := filepath.Join(tmpDir, "buildpack.toml")
			backupPath := filepath.Join(tmpDir, "buildpack.toml.bak")
			Expect(os.WriteFile(buildpackTOMLPath, []byte("placeholder = true\n"), 0600)).To(Succeed())
			Expect(os.WriteFile(backupPath, []byte(baseBuildpackTOML), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackIDs:            []string{"paketo-buildpacks/java@1.2.3"},
				SelectedVendors:         []string{"temurin"},
				BuildpackPath:           tmpDir,
				BuildpackTOMLPath:       buildpackTOMLPath,
				BuildpackTOMLBackupPath: backupPath,
				JVMVendors: []builder.JVMVendor{
					{
						BuildpackID: "paketo-buildpacks/temurin",
						Description: "Temurin buildpack",
						Homepage:    "https://adoptium.net",
						Name:        "Temurin",
						VendorID:    "temurin",
					},
				},
			}

			Expect(cmd.BuildMultipleBuildpacks()).To(HaveOccurred())
		})

		it("customizes buildpack.toml from backup for a selected vendor", func() {
			tmpDir := t.TempDir()
			buildpackPath := filepath.Join(tmpDir, "buildpack.toml")
			backupPath := filepath.Join(tmpDir, "buildpack.toml.bak")

			Expect(os.WriteFile(backupPath, []byte(`[buildpack]
id = "paketo-buildpacks/original"
name = "Original"
description = "Original description"
homepage = "https://original.example"
version = "0.0.1"

[[metadata.configurations]]
name = "BP_JVM_VENDORS"
default = "liberica,temurin"

[[metadata.dependencies]]
id = "jdk-liberica"

[[metadata.dependencies]]
id = "jdk-temurin"
`), 0600)).To(Succeed())

			Expect(os.WriteFile(buildpackPath, []byte("placeholder = true"), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				BuildpackTOMLPath:       buildpackPath,
				BuildpackTOMLBackupPath: backupPath,
			}

			vendor := builder.JVMVendor{
				BuildpackID: "paketo-buildpacks/temurin",
				Description: "Temurin buildpack",
				Homepage:    "https://adoptium.net",
				Name:        "Temurin",
				VendorID:    "temurin",
			}

			Expect(cmd.CustomizeBuildpackTOML(vendor, "9.9.9")).To(Succeed())

			contents, err := os.ReadFile(buildpackPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("id = \"paketo-buildpacks/temurin\""))
			Expect(string(contents)).To(ContainSubstring("version = \"9.9.9\""))
			Expect(string(contents)).To(ContainSubstring("default = \"temurin\""))
			Expect(string(contents)).To(ContainSubstring("id = \"jdk-temurin\""))
		})

		it("execute backs up and then runs single-buildpack flow", func() {
			t.Setenv("PATH", "")
			tmpDir := t.TempDir()
			buildpackTOMLPath := filepath.Join(tmpDir, "buildpack.toml")
			backupPath := filepath.Join(tmpDir, "buildpack.toml.bak")
			Expect(os.WriteFile(buildpackTOMLPath, []byte(baseBuildpackTOML), 0600)).To(Succeed())

			cmd := builder.BuildJvmVendorsCommand{
				SingleBuildpack:         true,
				BuildpackIDs:            []string{"paketo-buildpacks/java@1.2.3"},
				SelectedVendors:         []string{"temurin"},
				DefaultVendor:           "temurin",
				BuildpackPath:           tmpDir,
				BuildpackTOMLPath:       buildpackTOMLPath,
				BuildpackTOMLBackupPath: backupPath,
				JVMVendors: []builder.JVMVendor{
					{
						BuildpackID: "paketo-buildpacks/temurin",
						Description: "Temurin buildpack",
						Homepage:    "https://adoptium.net",
						Name:        "Temurin",
						VendorID:    "temurin",
					},
				},
			}

			Expect(cmd.Execute()).To(HaveOccurred())
			Expect(backupPath).To(BeARegularFile())
		})
	})
}
