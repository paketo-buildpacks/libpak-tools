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
package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"

	"github.com/paketo-buildpacks/libpak-tools/internal"
	"github.com/paketo-buildpacks/libpak-tools/packager"
)

type JVMVendor struct {
	Description string
	Homepage    string
	ID          string
	Name        string
}

type BuildJvmVendorsCommand struct {
	BuildpackID             string
	SingleBuildpack         bool
	AllVendors              bool
	SelectedVendors         []string
	BuildpackVersions       []string
	BuildpackPath           string
	BuildpackTOMLPath       string
	BuildpackTOMLBackupPath string
	CacheLocation           string
	IncludeDependencies     bool
	DependencyFilters       []string
	StrictDependencyFilters bool
	RegistryName            string
	Publish                 bool
	JVMVendors              []JVMVendor
}

// InferBuildpackPath infers the buildpack path from the buildpack id
func (b *BuildJvmVendorsCommand) InferBuildpackPath() error {
	root, found := os.LookupEnv("BP_ROOT")
	if !found {
		return fmt.Errorf("BP_ROOT must be set")
	}

	parts := strings.SplitN(b.BuildpackID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid buildpack id: %s, must contain two parts that are `/` separated", b.BuildpackID)
	}
	bpType, bpName := parts[0], parts[1]

	switch bpType {
	case "paketobuildpacks":
		b.BuildpackPath = filepath.Join(root, "paketo-buildpacks", bpName)
	case "paketocommunity":
		b.BuildpackPath = filepath.Join(root, "paketo-community", bpName)
	default:
		b.BuildpackPath = filepath.Join(root, b.BuildpackID)
	}

	return nil
}

// TODO: this needs test coverage
func (b *BuildJvmVendorsCommand) Execute() error {
	// make a backup copy of the original buildpack.toml
	if _, err := os.Stat(b.BuildpackTOMLBackupPath); err == nil {
		return fmt.Errorf("backup copy of original buildpack.toml already exists, a previous build must have failed, please reset buildpack directory")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for backup copy of original buildpack.toml: %w", err)
	}

	if err := sherpa.CopyFileFrom(b.BuildpackTOMLPath, b.BuildpackTOMLBackupPath); err != nil {
		return fmt.Errorf("failed to make backup copy of original buildpack.toml: %w", err)
	}

	if b.SingleBuildpack {
		fmt.Println("➜ Building single JVM Vendors buildpack")

		vendorList := []string{}
		for _, vendor := range b.SelectedVendors {
			parts := strings.SplitN(vendor, "/", 2)
			if len(parts) == 2 {
				vendorList = append(vendorList, parts[1])
			} else {
				vendorList = append(vendorList, vendor)
			}
		}

		if err := internal.UpdateTOMLFile(b.BuildpackTOMLPath, UpdateBuildpackConfiguration(map[string]interface{}{
			"BP_JVM_VENDORS": strings.Join(vendorList, ","),
			"BP_JVM_VENDOR":  "bellsoft-liberica",
		})); err != nil {
			return fmt.Errorf("failed to customize buildpack.toml: %w", err)
		}

		pkgCmd := packager.NewBundleBuildpack()
		pkgCmd.BuildpackID = b.BuildpackID
		pkgCmd.BuildpackPath = b.BuildpackPath
		pkgCmd.BuildpackVersion = b.BuildpackVersions[0]
		pkgCmd.CacheLocation = b.CacheLocation
		pkgCmd.IncludeDependencies = b.IncludeDependencies
		pkgCmd.DependencyFilters = b.DependencyFilters
		pkgCmd.StrictDependencyFilters = b.StrictDependencyFilters
		pkgCmd.RegistryName = b.RegistryName
		pkgCmd.Publish = b.Publish
		if err := pkgCmd.Execute(); err != nil {
			return fmt.Errorf("failed to build single JVM Vendors buildpack: %w", err)
		}
	} else {
		fmt.Println("➜ Building multiple JVM Vendors buildpacks")

		for i, vendor := range b.SelectedVendors {
			version := b.BuildpackVersions[i]
			selectedVendor := b.selectVendor(vendor)

			fmt.Printf("  Building %s@%s\n", vendor, version)

			if err := b.CustomizeBuildpackTOML(selectedVendor, version); err != nil {
				return fmt.Errorf("failed to customize buildpack.toml: %w", err)
			}

			pkgCmd := packager.NewBundleBuildpack()
			pkgCmd.BuildpackID = vendor
			pkgCmd.BuildpackPath = b.BuildpackPath
			pkgCmd.BuildpackVersion = b.BuildpackVersions[i]
			pkgCmd.CacheLocation = b.CacheLocation
			pkgCmd.IncludeDependencies = b.IncludeDependencies
			pkgCmd.DependencyFilters = b.DependencyFilters
			pkgCmd.StrictDependencyFilters = b.StrictDependencyFilters
			pkgCmd.RegistryName = b.RegistryName
			pkgCmd.Publish = b.Publish
			if err := pkgCmd.Execute(); err != nil {
				return err
			}
		}
	}

	// restore original buildpack.toml
	if err := sherpa.CopyFileFrom(b.BuildpackTOMLBackupPath, b.BuildpackTOMLPath); err != nil {
		return fmt.Errorf("failed to restore original buildpack.toml: %w", err)
	}

	return os.Remove(b.BuildpackTOMLBackupPath)
}

func (b *BuildJvmVendorsCommand) selectVendor(vendorID string) JVMVendor {
	var fullVendor JVMVendor
	for _, v := range b.JVMVendors {
		if v.ID == vendorID {
			fullVendor = v
			break
		}
	}
	return fullVendor
}

// CustomizeBuildpackTOML with the buildpack-specific information
//
// This needs to in-place for each buildpack, so we restore the original before copying
func (b *BuildJvmVendorsCommand) CustomizeBuildpackTOML(jvmVendor JVMVendor, version string) error {
	err := sherpa.CopyFileFrom(b.BuildpackTOMLBackupPath, b.BuildpackTOMLPath)
	if err != nil {
		return fmt.Errorf("failed to restore original buildpack.toml: %w", err)
	}

	parts := strings.SplitN(jvmVendor.ID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid buildpack id: %s, must contain two parts that are `/` separated", jvmVendor.ID)
	}

	return internal.MultiUpdateTOMLFILE(
		b.BuildpackTOMLPath,
		UpdateBuildpackDetails(jvmVendor, version),
		UpdateBuildpackConfiguration(map[string]interface{}{
			"BP_JVM_VENDORS": parts[1],
		}),
		RemoveDependenciesUnlessInVendorList([]string{parts[1]}))
}

// UpdateBuildpackDetails will get a full buildpack.toml and update the buildpack metadata with the provided details
func UpdateBuildpackDetails(jvmVendor JVMVendor, version string) func(map[string]interface{}) {
	return func(toml map[string]interface{}) {
		metadataRaw, found := toml["buildpack"]
		if !found {
			return
		}

		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			return
		}

		metadata["description"] = jvmVendor.Description
		metadata["homepage"] = jvmVendor.Homepage
		metadata["id"] = jvmVendor.ID
		metadata["name"] = jvmVendor.Name
		metadata["version"] = version
	}
}

// UpdateBuildpackConfiguration will get a full buildpack.toml and update the buildpack metadata configurations with the provided map
func UpdateBuildpackConfiguration(newConfigs map[string]interface{}) func(map[string]interface{}) {
	return func(toml map[string]interface{}) {
		metadataRaw, found := toml["metadata"]
		if !found {
			fmt.Println("metadata not found", toml)
			return
		}

		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			fmt.Println("metadata not a map", metadataRaw)
			return
		}

		configurationsRaw, found := metadata["configurations"]
		if !found {
			fmt.Println("configurations not found", metadata)
			return
		}

		configurations, ok := configurationsRaw.([]map[string]interface{})
		if !ok {
			fmt.Println("configurations not a map", configurationsRaw)
			return
		}

		for key, value := range newConfigs {
			for _, config := range configurations {
				if config["name"] == key {
					config["default"] = value
				}
			}
		}
	}
}

// RemoveDependenciesUnlessInVendorList will get a full buildpack.toml and remove all dependencies that are not in the provided list of vendors
func RemoveDependenciesUnlessInVendorList(vendors []string) func(map[string]interface{}) {
	return func(toml map[string]interface{}) {
		metadataRaw, found := toml["metadata"]
		if !found {
			fmt.Println("metadata not found", toml)
			return
		}

		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			fmt.Println("metadata not a map", metadataRaw)
			return
		}

		dependenciesRaw, found := metadata["dependencies"]
		if !found {
			fmt.Println("dependencies not found", metadata)
			return
		}

		dependencies, ok := dependenciesRaw.([]map[string]interface{})
		if !ok {
			fmt.Println("dependencies not a list", dependenciesRaw)
			return
		}

		newDeps := []map[string]interface{}{}
		for i, dep := range dependencies {
			depIDRaw, found := dep["id"]
			if !found {
				continue
			}

			depID, ok := depIDRaw.(string)
			if !ok {
				continue
			}

			found = false
			for _, vendor := range vendors {
				if strings.HasSuffix(depID, fmt.Sprintf("-%s", vendor)) {
					found = true
					break
				}
			}

			if found {
				newDeps = append(newDeps, dependencies[i])
			}
		}

		metadata["dependencies"] = newDeps
	}
}

// NOTES:
// 1. If we are building one big buildpack just build once with the stock buildpack.toml.
//
// 2. If we are building individual vendor buildpacks, then iterate the vendors list and for each buildpack to create filter the
//    list of metadata.dependencies[] in buildpack.toml to only include the dependencies for that vendor. Write a temporary buildpack.toml
//    with those dependencies and build that buildpack.
//
// UNKNOWNS:
//  - not sure how to handle setting versions of the buildpacks as we iterate through the vendors list??
//    - when building locally, you'll need to specify the versions for each buildpack to publish
//    - when building in CI, we'll need to infer the versions from the tag
//  - what is the workflow going to be like for CI?
//    - we can easily tag this repo, and then publish the all-vendors buildpack
//    - but what about individual vendor buildpacks? We need to tag those individually and publish them
//    - so we could probably use a tag prefix for this (bellsoft-liberica-vX.Y.Z, azul-zulu-vX.Y.Z, etc)
//    - that CI job would then trigger and set the vendor & version based on the tag information
