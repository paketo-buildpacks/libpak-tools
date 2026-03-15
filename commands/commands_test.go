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

package commands

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak-tools/builder"
	"github.com/paketo-buildpacks/libpak-tools/carton"
	"github.com/paketo-buildpacks/libpak-tools/internal/testutil"
	"github.com/paketo-buildpacks/libpak-tools/packager"
)

func testCommands(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("root command", func() {
		it("registers expected top-level subcommands", func() {
			subcommands := map[string]struct{}{}
			for _, cmd := range rootCmd.Commands() {
				subcommands[cmd.Name()] = struct{}{}
			}

			Expect(rootCmd.Use).To(Equal("libpak-tools"))
			Expect(rootCmd.Short).To(Equal("A set of tools for managing Paketo libpak based buildpacks"))
			Expect(subcommands).To(HaveKey("package"))
			Expect(subcommands).To(HaveKey("dependency"))
			Expect(subcommands).To(HaveKey("build-jvm-vendors"))
			Expect(subcommands).To(HaveKey("version"))
		})

		it("executes a valid subcommand", func() {
			previousVersion := version
			version = "test-version"
			defer func() {
				version = previousVersion
			}()

			rootCmd.SetArgs([]string{"version"})
			defer rootCmd.SetArgs(nil)

			stdout := testutil.CaptureStdout(t, func() {
				Execute()
			})

			Expect(stdout).To(ContainSubstring("libpak-tools test-version"))
		})
	})

	context("package command", func() {
		it("registers compile and bundle subcommands", func() {
			cmd := PackageCommand()
			subcommands := map[string]struct{}{}
			for _, sub := range cmd.Commands() {
				subcommands[sub.Name()] = struct{}{}
			}

			Expect(cmd.Use).To(Equal("package"))
			Expect(cmd.Short).To(Equal("Interact with packages"))
			Expect(cmd.Commands()).To(HaveLen(2))
			Expect(subcommands).To(HaveKey("compile"))
			Expect(subcommands).To(HaveKey("bundle"))
		})

		it("uses current working directory as default source for compile", func() {
			cmd := PackageCompileCommand()

			sourceFlag := cmd.Flags().Lookup("source")
			targetArchFlag := cmd.Flags().Lookup("target-arch")
			cwd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd.Use).To(Equal("compile"))
			Expect(sourceFlag).NotTo(BeNil())
			Expect(sourceFlag.DefValue).To(Equal(filepath.Clean(cwd)))
			Expect(targetArchFlag).NotTo(BeNil())
			Expect(targetArchFlag.DefValue).To(Equal("all"))
		})

		it("configures required bundle flags", func() {
			cmd := PackageBundleCommand()

			Expect(cmd.Use).To(Equal("bundle"))
			Expect(cmd.Flags().Lookup("buildpack-id")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("buildpack-path")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("version")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("registry-name")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("publish")).NotTo(BeNil())
		})
	})

	context("dependency command", func() {
		it("registers update subcommand hierarchy", func() {
			dependencyCmd := DependencyCommand()
			updateCmd, _, err := dependencyCmd.Find([]string{"update"})
			subcommands := map[string]struct{}{}
			for _, sub := range updateCmd.Commands() {
				subcommands[sub.Name()] = struct{}{}
			}

			Expect(err).NotTo(HaveOccurred())
			Expect(dependencyCmd.Use).To(Equal("dependency"))
			Expect(updateCmd.Use).To(Equal("update"))
			Expect(updateCmd.Commands()).To(HaveLen(4))
			Expect(subcommands).To(HaveKey("build-image"))
			Expect(subcommands).To(HaveKey("lifecycle"))
			Expect(subcommands).To(HaveKey("package"))
			Expect(subcommands).To(HaveKey("build-module"))
		})

		it("configures expected flags for build-module updates", func() {
			cmd := DependencyUpdateBuildModuleCommand()

			Expect(cmd.Use).To(Equal("build-module"))
			Expect(cmd.Flags().Lookup("buildmodule-toml")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("id")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("sha256")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("uri")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("version")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("version-pattern")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("purl")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("cpe")).NotTo(BeNil())
		})
	})

	context("version and JVM vendor commands", func() {
		it("prints the configured version", func() {
			previousVersion := version
			version = "2.3.4"
			defer func() {
				version = previousVersion
			}()

			cmd := VersionCommand()
			output := testutil.CaptureStdout(t, func() {
				cmd.Run(cmd, nil)
			})

			Expect(output).To(ContainSubstring("libpak-tools 2.3.4"))
		})

		it("sets vendor selection flags for build-jvm-vendors", func() {
			cmd := BuildJvmVendorsCommand()

			Expect(cmd.Use).To(Equal("build-jvm-vendors"))
			Expect(cmd.Flags().Lookup("vendors")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("include-all-vendors")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("buildpack-id")).NotTo(BeNil())
			Expect(cmd.Flags().Lookup("single-buildpack")).NotTo(BeNil())
		})
	})

	context("build JVM vendor helper", func() {
		var (
			jvmVendorList = []builder.JVMVendor{
				{VendorID: "temurin", Default: true},
				{VendorID: "liberica"},
			}
			allVendors = []string{"temurin", "liberica"}
		)

		it("rejects empty vendor selection when include-all is false", func() {
			i := builder.BuildJvmVendorsCommand{}
			err := runBuildJvmVendorsCommand(&i, jvmVendorList, allVendors)
			Expect(err).To(MatchError("--vendors must be set or --include-all-vendors must be set"))
		})

		it("requires single-buildpack when include-all-vendors is set", func() {
			i := builder.BuildJvmVendorsCommand{
				AllVendors:      true,
				SelectedVendors: []string{"temurin"},
			}
			err := runBuildJvmVendorsCommand(&i, jvmVendorList, allVendors)
			Expect(err).To(MatchError("--include-all-vendors can only be used with --single-buildpack"))
		})

		it("rejects unknown vendors", func() {
			i := builder.BuildJvmVendorsCommand{
				SelectedVendors: []string{"does-not-exist"},
				BuildpackPath:   "/tmp",
			}
			err := runBuildJvmVendorsCommand(&i, jvmVendorList, allVendors)
			Expect(err).To(MatchError(ContainSubstring("invalid vendor")))
		})

		it("defaults to all vendors and returns a wrapped execute error", func() {
			i := builder.BuildJvmVendorsCommand{
				AllVendors:      true,
				SingleBuildpack: true,
				BuildpackIDs:    []string{"paketo-buildpacks/java@1.2.3"},
				BuildpackPath:   t.TempDir(),
			}
			err := runBuildJvmVendorsCommand(&i, jvmVendorList, allVendors)
			Expect(err).To(MatchError(ContainSubstring("JVM Vendors build failed")))
			Expect(i.SelectedVendors).To(ConsistOf("temurin", "liberica"))
		})
	})

	context("build module update helper", func() {
		it("validates required inputs", func() {
			b := carton.BuildModuleDependency{}
			err := runDependencyUpdateBuildModule(&b)
			Expect(err).To(MatchError("buildmodule toml path must be set"))
		})

		it("applies default arch/purl/cpe before update", func() {
			buildmodulePath := filepath.Join(t.TempDir(), "buildpack.toml")
			Expect(os.WriteFile(buildmodulePath, []byte(`[[metadata.dependencies]]
id = "test-jre"
version = "1.0.0"
uri = "https://example.com/old"
sha256 = "old-sha"
purl = "pkg:generic/test-jre@1.0.0?arch=amd64"
cpes = ["cpe:2.3:a:test-vendor:test-product:1.0.0:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

			b := carton.BuildModuleDependency{
				BuildModulePath: buildmodulePath,
				ID:              "test-jre",
				SHA256:          "new-sha",
				URI:             "https://example.com/new",
				Version:         "2.0.0",
				VersionPattern:  "1.0.0",
			}

			Expect(runDependencyUpdateBuildModule(&b)).To(Succeed())
			Expect(b.Arch).To(Equal("amd64"))
			Expect(b.PURL).To(Equal("2.0.0"))
			Expect(b.PURLPattern).To(Equal("1.0.0"))
			Expect(b.CPE).To(Equal("2.0.0"))
			Expect(b.CPEPattern).To(Equal("1.0.0"))
		})
	})

	context("package bundle helper", func() {
		it("requires buildpack id or path", func() {
			p := packager.NewBundleBuildpack()
			err := runPackageBundleCommand(&p)
			Expect(err).To(MatchError("buildpack-id or buildpack-path must be set"))
		})

		it("requires buildpack id when path is set", func() {
			p := packager.NewBundleBuildpack()
			p.BuildpackPath = t.TempDir()
			err := runPackageBundleCommand(&p)
			Expect(err).To(MatchError("buildpack-id and buildpack-path must both be set"))
		})

		it("defaults registry name to buildpack id", func() {
			buildpackPath := t.TempDir()
			Expect(os.WriteFile(filepath.Join(buildpackPath, "buildpack.toml"), []byte(`[buildpack]
id = "some-id"
version = "1.0.0"
name = "Some Buildpack"
description = "A buildpack for testing"
homepage = "https://example.com"
`), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(buildpackPath, "package.toml"), []byte("[metadata]\n"), 0600)).To(Succeed())

			p := packager.NewBundleBuildpack()
			p.BuildpackID = "some-id"
			p.BuildpackPath = buildpackPath
			p.BuildpackVersion = "1.0.0"

			Expect(runPackageBundleCommand(&p)).To(Succeed())
			Expect(p.RegistryName).To(Equal("some-id"))
		})
	})
}
