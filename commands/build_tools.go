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
	_ "embed"
	"fmt"
	"log"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/paketo-buildpacks/libpak-tools/builder"
)

//go:embed build_tools.toml
var BuildToolsTOML []byte

func BuildBuildToolsCommand() *cobra.Command {
	buildToolData := struct{ Tools []builder.BuildTool }{}
	if err := toml.Unmarshal(BuildToolsTOML, &buildToolData); err != nil {
		log.Fatalf("unable to decode build tools list\n%s", err)
	}
	// work around TOML not allowing top-level arrays
	buildToolList := buildToolData.Tools

	allTools := []string{}
	for _, tool := range buildToolList {
		allTools = append(allTools, tool.ToolID)
	}

	i := builder.BuildBuildToolsCommand{}

	var buildBuildToolsCommand = &cobra.Command{
		Use:   "build-build-tools",
		Short: "Build Build Tools Buildpacks",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runBuildBuildToolsCommand(&i, buildToolList, allTools); err != nil {
				log.Fatal(err)
			}
		},
	}

	buildBuildToolsCommand.Flags().StringArrayVar(&i.BuildpackIDs, "buildpack-id", []string{}, "buildpack id and version in the format 'id@version'")
	buildBuildToolsCommand.Flags().BoolVar(&i.SingleBuildpack, "single-buildpack", false, "build output is a single buildpack with the selected tools (default: false)")
	buildBuildToolsCommand.Flags().BoolVar(&i.AllTools, "include-all-tools", false, "include all of the tools (default: false)")
	buildBuildToolsCommand.Flags().StringArrayVar(&i.SelectedTools, "tools", []string{}, "list of tools to build")
	buildBuildToolsCommand.Flags().StringVar(&i.BuildpackPath, "buildpack-path", "", "path to build-tools buildpack directory")
	buildBuildToolsCommand.Flags().StringVar(&i.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	buildBuildToolsCommand.Flags().BoolVar(&i.IncludeDependencies, "include-dependencies", false, "whether to include dependencies, applies to all buildpacks (default: false)")
	buildBuildToolsCommand.Flags().StringArrayVar(&i.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies, applies to all buildpacks")
	buildBuildToolsCommand.Flags().BoolVar(&i.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data, applies to all buildpacks (default: false)")
	buildBuildToolsCommand.Flags().StringVar(&i.RegistryName, "registry-name", "", "prefix for the registry to publish to, applies to all buildpacks (default: the buildpack id)")
	buildBuildToolsCommand.Flags().BoolVar(&i.Publish, "publish", false, "publish the buildpack to a buildpack registry, applies to all buildpacks (default: false)")
	buildBuildToolsCommand.Flags().StringVar(&i.Format, "format", "image", "format for the output, either image or file (default: image)")
	buildBuildToolsCommand.Flags().StringVar(&i.OutputPath, "output-path", "", "path to directory where output files will be created (default: current directory)")

	return buildBuildToolsCommand
}

func runBuildBuildToolsCommand(i *builder.BuildBuildToolsCommand, buildToolList []builder.BuildTool, allTools []string) error {
	i.Tools = buildToolList

	if len(i.BuildpackIDs) == 0 {
		log.Printf("No buildpack IDs specified, you must specify one buildpack ID if using --single-buildpack or a single buildpack will be built per buildpack ID specified")
	}

	if len(i.SelectedTools) == 0 && !i.AllTools {
		return fmt.Errorf("--tools must be set or --include-all-tools must be set")
	}

	if len(i.SelectedTools) > 0 && i.AllTools {
		log.Printf("Warning: both --tools and --include-all-tools flags are set, ignoring --tools flags")
	}

	if i.AllTools && !i.SingleBuildpack {
		return fmt.Errorf("--include-all-tools can only be used with --single-buildpack")
	}

	if i.SingleBuildpack && len(i.BuildpackIDs) != 1 {
		return fmt.Errorf("--single-buildpack requires one single and only one --buildpack-id to be specified, but got %q", i.BuildpackIDs)
	}

	if i.AllTools {
		i.SelectedTools = allTools
	} else {
		for _, tool := range i.SelectedTools {
			if !slices.Contains(allTools, tool) {
				return fmt.Errorf("invalid tool: %s, possible tools are %q", tool, allTools)
			}
		}
	}

	if i.BuildpackPath == "" && (!i.SingleBuildpack || len(i.BuildpackIDs) > 1) {
		return fmt.Errorf("you must specify --buildpack-path when building multiple buildpacks")
	}

	if i.BuildpackPath == "" {
		if err := i.InferBuildpackPath(); err != nil {
			return err
		}
	}

	i.BuildpackTOMLPath = filepath.Join(i.BuildpackPath, "buildpack.toml")
	i.BuildpackTOMLBackupPath = filepath.Join(i.BuildpackPath, "buildpack.toml.bak")

	if err := i.Execute(); err != nil {
		return fmt.Errorf("build tools build failed\n%w", err)
	}

	return nil
}
