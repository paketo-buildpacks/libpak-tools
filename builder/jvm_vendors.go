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
	"regexp"
	"slices"
	"strings"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"

	"github.com/paketo-buildpacks/libpak-tools/internal"
	"github.com/paketo-buildpacks/libpak-tools/packager"
)

type JVMVendor struct {
	BuildpackID string `toml:"buildpack_id"`
	Default     bool   `toml:"default"`
	Description string `toml:"description"`
	Homepage    string `toml:"homepage"`
	Name        string `toml:"name"`
	VendorID    string `toml:"vendor_id"`
}

type BuildJvmVendorsCommand struct {
	BuildpackIDs            []string
	SingleBuildpack         bool
	AllVendors              bool
	SelectedVendors         []string
	DefaultVendor           string
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

	if len(b.BuildpackIDs) == 0 {
		return fmt.Errorf("no buildpack IDs specified, cannot infer buildpack path")
	}

	// Named capture groups for better parsing
	re := regexp.MustCompile(`^(?P<org>[a-zA-Z0-9-_]+)/(?P<name>[a-zA-Z0-9-_]+)@(?P<version>\d+\.\d+\.\d+)$`)

	matches := re.FindStringSubmatch(b.BuildpackIDs[0])
	if matches == nil {
		return fmt.Errorf("invalid buildpack id: %s, must match format 'org/name@version'", b.BuildpackIDs[0])
	}

	bpType := matches[1]
	bpName := matches[2]

	switch bpType {
	case "paketo-buildpacks":
		b.BuildpackPath = filepath.Join(root, "paketo-buildpacks", bpName)
	case "paketobuildpacks":
		b.BuildpackPath = filepath.Join(root, "paketo-buildpacks", bpName)
	case "paketo-community":
		b.BuildpackPath = filepath.Join(root, "paketo-community", bpName)
	case "paketocommunity":
		b.BuildpackPath = filepath.Join(root, "paketo-community", bpName)
	default:
		b.BuildpackPath = filepath.Join(root, b.BuildpackIDs[0])
	}

	return nil
}

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
		if err := b.BuildSingleBuildpack(); err != nil {
			return fmt.Errorf("failed to build single buildpack: %w", err)
		}
	} else {
		if err := b.BuildMultipleBuildpacks(); err != nil {
			return fmt.Errorf("failed to build multiple buildpacks: %w", err)
		}
	}

	// restore original buildpack.toml
	if err := sherpa.CopyFileFrom(b.BuildpackTOMLBackupPath, b.BuildpackTOMLPath); err != nil {
		return fmt.Errorf("failed to restore original buildpack.toml: %w", err)
	}

	return os.Remove(b.BuildpackTOMLBackupPath)
}

// BuildSingleBuildpack builds a single buildpack from the list of JVM Vendors
func (b *BuildJvmVendorsCommand) BuildSingleBuildpack() error {
	fmt.Println("➜ Building single JVM Vendors buildpack")

	if len(b.BuildpackIDs) == 0 {
		return fmt.Errorf("no buildpack IDs specified, you must specify one buildpack ID if using --single-buildpack")
	}

	if len(b.BuildpackIDs) > 1 {
		return fmt.Errorf("single buildpack requires exactly one buildpack ID, got %q", b.BuildpackIDs)
	}

	parts := strings.SplitN(b.BuildpackIDs[0], "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid buildpack ID: %s, must contain two parts that are `@` separated", b.BuildpackIDs[0])
	}

	defaultVendorId, err := b.selectDefaultVendor()
	if err != nil {
		return fmt.Errorf("unable to select default vendor: %w", err)
	}

	if err := internal.UpdateTOMLFile(b.BuildpackTOMLPath, UpdateBuildpackConfiguration(map[string]interface{}{
		"BP_JVM_VENDORS": strings.Join(b.SelectedVendors, ","),
		"BP_JVM_VENDOR":  defaultVendorId,
	})); err != nil {
		return fmt.Errorf("failed to customize buildpack.toml: %w", err)
	}

	pkgCmd := packager.NewBundleBuildpack()
	pkgCmd.BuildpackID = parts[0]
	pkgCmd.BuildpackPath = b.BuildpackPath
	pkgCmd.BuildpackVersion = parts[1]
	pkgCmd.CacheLocation = b.CacheLocation
	pkgCmd.IncludeDependencies = b.IncludeDependencies
	pkgCmd.DependencyFilters = b.DependencyFilters
	pkgCmd.StrictDependencyFilters = b.StrictDependencyFilters
	pkgCmd.RegistryName = b.RegistryName
	pkgCmd.Publish = b.Publish
	return pkgCmd.Execute()
}

// BuildMultipleBuildpacks builds multiple buildpacks one with each JVM Vendor
func (b *BuildJvmVendorsCommand) BuildMultipleBuildpacks() error {
	fmt.Println("➜ Building multiple JVM Vendors buildpacks")

	if len(b.BuildpackIDs) != len(b.SelectedVendors) {
		return fmt.Errorf("number of buildpack IDs (%q) must match number of selected vendors (%q)", b.BuildpackIDs, b.SelectedVendors)
	}

	for i, buildpackID := range b.BuildpackIDs {
		parts := strings.SplitN(buildpackID, "@", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid buildpack ID: %s, must contain two parts that are `@` separated", buildpackID)
		}

		selectedVendor := b.selectVendor(b.SelectedVendors[i])

		fmt.Printf("  Building %s\n", buildpackID)

		if err := b.CustomizeBuildpackTOML(selectedVendor, parts[1]); err != nil {
			return fmt.Errorf("failed to customize buildpack.toml: %w", err)
		}

		pkgCmd := packager.NewBundleBuildpack()
		pkgCmd.BuildpackID = parts[0]
		pkgCmd.BuildpackPath = b.BuildpackPath
		pkgCmd.BuildpackVersion = parts[1]
		pkgCmd.CacheLocation = b.CacheLocation
		pkgCmd.IncludeDependencies = b.IncludeDependencies
		pkgCmd.DependencyFilters = b.DependencyFilters
		pkgCmd.StrictDependencyFilters = b.StrictDependencyFilters
		pkgCmd.RegistryName = b.RegistryName
		pkgCmd.Publish = b.Publish
		pkgCmd.SkipClean = i < len(b.BuildpackIDs)-1 // Skip clean on the last buildpack to avoid cleaning up resources needed for subsequent builds
		if err := pkgCmd.Execute(); err != nil {
			return err
		}

		fmt.Println()
	}

	return nil
}

func (b *BuildJvmVendorsCommand) selectVendor(vendorID string) JVMVendor {
	var fullVendor JVMVendor
	for _, v := range b.JVMVendors {
		if v.VendorID == vendorID {
			fullVendor = v
			break
		}
	}
	return fullVendor
}

func (b *BuildJvmVendorsCommand) selectDefaultVendor() (string, error) {
	var defaultVendor string

	for _, vendor := range b.SelectedVendors {
		if vendor == b.DefaultVendor {
			defaultVendor = vendor
			break
		}
	}

	if defaultVendor == "" {
		for _, v := range b.JVMVendors {
			if v.Default && slices.Contains(b.SelectedVendors, v.VendorID) {
				defaultVendor = v.VendorID
				break
			}
		}
	}

	if defaultVendor == "" {
		if len(b.JVMVendors) > 0 {
			defaultVendor = b.SelectedVendors[0]
		}
	}

	if defaultVendor == "" {
		return "", fmt.Errorf("no default vendor specified via cli args, and no default vendor found in the list of vendors")
	}

	fmt.Println("➜ Using default vendor", defaultVendor, "from", b.SelectedVendors)
	return defaultVendor, nil
}

// CustomizeBuildpackTOML with the buildpack-specific information
//
// This needs to in-place for each buildpack, so we restore the original before copying
func (b *BuildJvmVendorsCommand) CustomizeBuildpackTOML(jvmVendor JVMVendor, version string) error {
	err := sherpa.CopyFileFrom(b.BuildpackTOMLBackupPath, b.BuildpackTOMLPath)
	if err != nil {
		return fmt.Errorf("failed to restore original buildpack.toml: %w", err)
	}

	return internal.MultiUpdateTOMLFILE(
		b.BuildpackTOMLPath,
		UpdateBuildpackDetails(jvmVendor, version),
		UpdateBuildpackConfiguration(map[string]interface{}{
			"BP_JVM_VENDORS": jvmVendor.VendorID,
		}),
		RemoveDependenciesUnlessInVendorList([]string{jvmVendor.VendorID}))
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
		metadata["id"] = jvmVendor.BuildpackID
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
