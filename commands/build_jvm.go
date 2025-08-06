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
		allVendors = append(allVendors, vendor.VendorID)
	}

	i := builder.BuildJvmVendorsCommand{}

	var buildJvmVendorsCommand = &cobra.Command{
		Use:   "build-jvm-vendors",
		Short: "Build JVM Vendors Buildpacks",
		Run: func(cmd *cobra.Command, args []string) {
			i.JVMVendors = jvmVendorList

			if len(i.BuildpackIDs) == 0 {
				log.Printf("No buildpack IDs specified, you must specify one buildpack ID if using --single-buildpack or a single buildpack will be built per buildpack ID specified")
			}

			if len(i.SelectedVendors) == 0 && !i.AllVendors {
				log.Fatal("--vendors must be set or --include-all-vendors must be set")
			}

			if len(i.SelectedVendors) > 0 && i.AllVendors {
				log.Printf("Warning: both --buildpack and --include-all-vendors flags are set, ignoring --buildpacks flags")
			}

			if i.AllVendors && !i.SingleBuildpack {
				log.Fatal("--include-all-vendors can only be used with --single-buildpack")
			}

			if i.SingleBuildpack && len(i.BuildpackIDs) != 1 {
				log.Fatalf("--single-buildpack requires one single and only one --buildpack-id to be specified, but got %q", i.BuildpackIDs)
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

			if i.BuildpackPath == "" && (!i.SingleBuildpack || len(i.BuildpackIDs) > 1) {
				log.Fatal("You must specify --buildpack-path when building multiple buildpacks")
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

	buildJvmVendorsCommand.Flags().StringArrayVar(&i.BuildpackIDs, "buildpack-id", []string{}, "buildpack id and version in the format 'id@version' (default: all vendors)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.SingleBuildpack, "single-buildpack", false, "build output is a single buildpack with listed vendors (default: false)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.AllVendors, "include-all-vendors", false, "include all of the vendors (default: false)")
	buildJvmVendorsCommand.Flags().StringArrayVar(&i.SelectedVendors, "vendors", []string{}, "list of vendors to build")
	buildJvmVendorsCommand.Flags().StringVar(&i.DefaultVendor, "default-vendor", "", "default vendor to use, if not set the the configured default vendor or first in the vendor list will be used")
	buildJvmVendorsCommand.Flags().StringVar(&i.BuildpackPath, "buildpack-path", "", "path to jvm-vendors buildpack directory")
	buildJvmVendorsCommand.Flags().StringVar(&i.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.IncludeDependencies, "include-dependencies", false, "whether to include dependencies, applies to all buildpacks (default: false)")
	buildJvmVendorsCommand.Flags().StringArrayVar(&i.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies, applies to all buildpacks")
	buildJvmVendorsCommand.Flags().BoolVar(&i.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data, applies to all buildpacks (default: false)")
	buildJvmVendorsCommand.Flags().StringVar(&i.RegistryName, "registry-name", "", "prefix for the registry to publish to, applies to all buildpacks (default: the buildpack id)")
	buildJvmVendorsCommand.Flags().BoolVar(&i.Publish, "publish", false, "publish the buildpack to a buildpack registry, applies to all buildpacks (default: false)")

	return buildJvmVendorsCommand
}
