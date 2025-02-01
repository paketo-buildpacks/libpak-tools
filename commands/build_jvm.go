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

package commands

import (
	_ "embed"
	"log"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/paketo-buildpacks/libpak-tools/builder"
)

//go:embed jvm_vendors.toml
var JVMVendorsTOML []byte

func BuildJvmVendorsCommand() *cobra.Command {
	jvmVendorData := struct{ Vendors []builder.JVMVendor }{}
	if err := toml.Unmarshal(JVMVendorsTOML, &jvmVendorData); err != nil {
		log.Fatalf("unable to decode jvm vendors list\n%s", err)
	}
	// work around TOML not allowing top-level arrays
	jvmVendorList := jvmVendorData.Vendors

	allVendors := []string{}
	for _, vendor := range jvmVendorList {
		allVendors = append(allVendors, vendor.ID)
	}

	i := builder.BuildJvmVendorsCommand{}

	var buildJvmVendorsCommand = &cobra.Command{
		Use:   "build-jvm-vendors",
		Short: "Build JVM Vendors Buildpacks",
		Run: func(cmd *cobra.Command, args []string) {
			i.JVMVendors = jvmVendorList

			if len(i.SelectedVendors) == 0 && !i.AllVendors {
				log.Fatal("vendors must be set or include-all-buildpacks must be set")
			}

			if len(i.SelectedVendors) > 0 && i.AllVendors {
				log.Printf("Warning: both --buildpack and --include-all-buildpacks flags are set, ignoring --buildpacks flags")
			}

			if i.AllVendors {
				i.SelectedVendors = allVendors
			} else {
				for _, vendor := range i.SelectedVendors {
					if !slices.Contains(allVendors, vendor) {
						log.Fatalf("Invalid vendor: %s, possible vendors are %q\n", vendor, allVendors)
					}
				}
			}

			if i.SingleBuildpack && len(i.BuildpackVersions) != 1 {
				log.Fatalf("Single buildpack requires exactly one version, got %q\n", i.BuildpackVersions)
			}

			if !i.SingleBuildpack && len(i.BuildpackVersions) != len(i.SelectedVendors) {
				log.Fatalf("Number of versions must match the number of vendors, got %q versions and %q vendors\n", i.BuildpackVersions, i.SelectedVendors)
			}

			if i.BuildpackPath == "" {
				if err := i.InferBuildpackPath(); err != nil {
					log.Fatal(err)
				}
			}

			i.BuildpackTOMLPath = filepath.Join(i.BuildpackPath, "buildpack.toml")
			i.BuildpackTOMLBackupPath = filepath.Join(i.BuildpackPath, "buildpack.toml.bak")

			err := i.Execute()
			if err != nil {
				log.Fatal("JVM Vendors build failed\n", err)
			}
		},
	}

	buildJvmVendorsCommand.Flags().StringVar(&i.BuildpackID, "buildpack-id", "paketo-buildpacks/jvm-vendors", "buildpack id")
	buildJvmVendorsCommand.Flags().BoolVar(&i.SingleBuildpack, "single-buildpack", false, "build output is a single buildpack with listed vendors (default: false)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.AllVendors, "include-all-vendors", false, "include all of the vendors (default: false)")
	buildJvmVendorsCommand.Flags().StringArrayVar(&i.SelectedVendors, "vendors", []string{}, "list of vendors to build")
	buildJvmVendorsCommand.Flags().StringArrayVar(&i.BuildpackVersions, "version", []string{}, "versions to substitute into buildpack.toml/extension.toml")
	buildJvmVendorsCommand.Flags().StringVar(&i.BuildpackPath, "buildpack-path", "", "path to buildpack directory")
	buildJvmVendorsCommand.Flags().StringVar(&i.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.IncludeDependencies, "include-dependencies", false, "whether to include dependencies (default: false)")
	buildJvmVendorsCommand.Flags().StringArrayVar(&i.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies")
	buildJvmVendorsCommand.Flags().BoolVar(&i.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data (default: false)")
	buildJvmVendorsCommand.Flags().StringVar(&i.RegistryName, "registry-name", "", "prefix for the registry to publish to (default: the buildpack id)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.Publish, "publish", false, "publish the buildpack to a buildpack registry (default: false)")

	return buildJvmVendorsCommand
}
