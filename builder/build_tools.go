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

// BuildTool describes one build tool that can be emitted as its own buildpack
// from the shared build-tools source.
type BuildTool struct {
	BuildpackID  string   `toml:"buildpack_id"`
	Default      bool     `toml:"default"`
	Description  string   `toml:"description"`
	Homepage     string   `toml:"homepage"`
	Name         string   `toml:"name"`
	ToolID       string   `toml:"tool_id"`
	Dependencies []string `toml:"dependencies"`
}

type BuildBuildToolsCommand struct {
	BuildpackIDs            []string
	SingleBuildpack         bool
	AllTools                bool
	SelectedTools           []string
	BuildpackPath           string
	BuildpackTOMLPath       string
	BuildpackTOMLBackupPath string
	CacheLocation           string
	IncludeDependencies     bool
	DependencyFilters       []string
	StrictDependencyFilters bool
	RegistryName            string
	Publish                 bool
	Format                  string
	OutputPath              string
	Tools                   []BuildTool
}

// InferBuildpackPath infers the buildpack path from the buildpack id
func (b *BuildBuildToolsCommand) InferBuildpackPath() error {
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

func (b *BuildBuildToolsCommand) Execute() error {
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

// BuildSingleBuildpack builds a single buildpack from the list of build tools.
func (b *BuildBuildToolsCommand) BuildSingleBuildpack() error {
	fmt.Println("➜ Building single build-tools buildpack")

	if len(b.BuildpackIDs) != 1 {
		return fmt.Errorf("single buildpack requires exactly one buildpack ID, got %q", b.BuildpackIDs)
	}

	id, version, err := splitBuildpackID(b.BuildpackIDs[0])
	if err != nil {
		return err
	}

	if err := internal.UpdateTOMLFile(b.BuildpackTOMLPath, UpdateBuildpackConfiguration(map[string]interface{}{
		"BP_BUILD_TOOLS": strings.Join(b.SelectedTools, ","),
	})); err != nil {
		return fmt.Errorf("failed to customize buildpack.toml: %w", err)
	}

	return b.packageBuildpack(id, version, "build-tools.cnb", true)
}

// BuildMultipleBuildpacks builds one buildpack per selected build tool.
func (b *BuildBuildToolsCommand) BuildMultipleBuildpacks() error {
	fmt.Println("➜ Building individual build-tools buildpacks")

	if len(b.BuildpackIDs) != len(b.SelectedTools) {
		return fmt.Errorf("number of buildpack IDs (%q) must match number of selected tools (%q)", b.BuildpackIDs, b.SelectedTools)
	}

	for i, buildpackID := range b.BuildpackIDs {
		id, version, err := splitBuildpackID(buildpackID)
		if err != nil {
			return err
		}

		selectedTool := b.selectTool(b.SelectedTools[i])

		fmt.Printf("  Building %s\n", buildpackID)

		if err := b.CustomizeBuildpackTOML(selectedTool, version); err != nil {
			return fmt.Errorf("failed to customize buildpack.toml: %w", err)
		}

		if err := b.packageBuildpack(id, version, fmt.Sprintf("%s.cnb", selectedTool.ToolID), i < len(b.BuildpackIDs)-1); err != nil {
			return err
		}

		fmt.Println()
	}

	return nil
}

func (b *BuildBuildToolsCommand) packageBuildpack(id, version, outputName string, skipClean bool) error {
	pkgCmd := packager.NewBundleBuildpack()
	pkgCmd.BuildpackID = id
	pkgCmd.BuildpackPath = b.BuildpackPath
	pkgCmd.BuildpackVersion = version
	pkgCmd.CacheLocation = b.CacheLocation
	pkgCmd.IncludeDependencies = b.IncludeDependencies
	pkgCmd.DependencyFilters = b.DependencyFilters
	pkgCmd.StrictDependencyFilters = b.StrictDependencyFilters
	pkgCmd.RegistryName = b.RegistryName
	pkgCmd.Publish = b.Publish
	pkgCmd.Format = b.Format
	pkgCmd.OutputName = outputName
	pkgCmd.OutputPath = b.OutputPath
	pkgCmd.SkipClean = b.Format == "file" || skipClean

	return pkgCmd.Execute()
}

// CustomizeBuildpackTOML updates the shared buildpack.toml with the details for
// one tool and removes the dependencies that the tool does not use.
//
// This needs to happen in-place for each buildpack, so we restore the original
// before each update.
func (b *BuildBuildToolsCommand) CustomizeBuildpackTOML(tool BuildTool, version string) error {
	if err := sherpa.CopyFileFrom(b.BuildpackTOMLBackupPath, b.BuildpackTOMLPath); err != nil {
		return fmt.Errorf("failed to restore original buildpack.toml: %w", err)
	}

	return internal.MultiUpdateTOMLFILE(
		b.BuildpackTOMLPath,
		UpdateBuildpackDetailsForTool(tool, version),
		UpdateBuildpackConfiguration(map[string]interface{}{
			"BP_BUILD_TOOLS": tool.ToolID,
		}),
		KeepDependencies(tool.Dependencies))
}

func (b *BuildBuildToolsCommand) selectTool(toolID string) BuildTool {
	for _, tool := range b.Tools {
		if tool.ToolID == toolID {
			return tool
		}
	}

	return BuildTool{}
}

// UpdateBuildpackDetailsForTool updates the buildpack metadata with the details
// for a single tool.
func UpdateBuildpackDetailsForTool(tool BuildTool, version string) func(map[string]interface{}) {
	return func(toml map[string]interface{}) {
		metadataRaw, found := toml["buildpack"]
		if !found {
			return
		}

		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			return
		}

		metadata["description"] = tool.Description
		metadata["homepage"] = tool.Homepage
		metadata["id"] = tool.BuildpackID
		metadata["name"] = tool.Name
		metadata["version"] = version
	}
}

// KeepDependencies removes every dependency and dependency constraint whose ID
// is not in the provided list.
func KeepDependencies(ids []string) func(map[string]interface{}) {
	return func(toml map[string]interface{}) {
		metadataRaw, found := toml["metadata"]
		if !found {
			return
		}

		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			return
		}

		if dependencies, ok := metadata["dependencies"].([]map[string]interface{}); ok {
			newDeps := []map[string]interface{}{}
			for _, dependency := range dependencies {
				if depID, ok := dependency["id"].(string); ok && slices.Contains(ids, depID) {
					newDeps = append(newDeps, dependency)
				}
			}
			metadata["dependencies"] = newDeps
		}

		if constraints, ok := metadata["dependency-constraints"].([]map[string]interface{}); ok {
			newConstraints := []map[string]interface{}{}
			for _, constraint := range constraints {
				if depID, ok := constraint["id"].(string); ok && slices.Contains(ids, depID) {
					newConstraints = append(newConstraints, constraint)
				}
			}
			metadata["dependency-constraints"] = newConstraints
		}
	}
}

func splitBuildpackID(buildpackID string) (id string, version string, err error) {
	parts := strings.SplitN(buildpackID, "@", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid buildpack ID: %s, must contain two parts that are `@` separated", buildpackID)
	}

	return parts[0], parts[1], nil
}
