/*
 * Copyright 2018-2024 the original author or authors.
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
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/paketo-buildpacks/libpak-tools/packager"
)

func PackageBundleCommand() *cobra.Command {
	p := packager.NewBundleBuildpack()

	var packageBuildpackCmd = &cobra.Command{
		Use:   "bundle",
		Short: "Compile and package a single buildpack (component & composite)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runPackageBundleCommand(&p); err != nil {
				log.Fatal(err)
			}
		},
	}

	packageBuildpackCmd.Flags().StringVar(&p.BuildpackID, "buildpack-id", "", "id of the buildpack to use")
	packageBuildpackCmd.Flags().StringVar(&p.BuildpackPath, "buildpack-path", "", "path to buildpack directory")
	packageBuildpackCmd.Flags().StringVar(&p.BuildpackVersion, "version", "", "version to substitute into buildpack.toml/extension.toml")
	packageBuildpackCmd.Flags().StringVar(&p.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	packageBuildpackCmd.Flags().BoolVar(&p.IncludeDependencies, "include-dependencies", false, "whether to include dependencies (default: false)")
	packageBuildpackCmd.Flags().StringArrayVar(&p.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies")
	packageBuildpackCmd.Flags().BoolVar(&p.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data (default: false)")
	packageBuildpackCmd.Flags().StringVar(&p.RegistryName, "registry-name", "", "prefix for the registry to publish to (default: your buildpack id)")
	packageBuildpackCmd.Flags().BoolVar(&p.Publish, "publish", false, "publish the buildpack to a buildpack registry (default: false)")
	packageBuildpackCmd.Flags().StringVar(&p.Format, "format", "image", "format for the output, either image or file (default: image)")
	packageBuildpackCmd.Flags().StringVar(&p.OutputName, "output-name", "buildpackage.cnb", "name of the output file when format is set to file (default: buildpackage.cnb)")

	return packageBuildpackCmd
}

func runPackageBundleCommand(p *packager.BundleBuildpack) error {
	if p.BuildpackID == "" && p.BuildpackPath == "" {
		return fmt.Errorf("buildpack-id or buildpack-path must be set")
	}

	if p.BuildpackPath != "" && p.BuildpackID == "" {
		return fmt.Errorf("buildpack-id and buildpack-path must both be set")
	}

	if p.Format != "image" && p.Format != "file" {
		return fmt.Errorf("invalid format %s, must be either image or file", p.Format)
	}

	if p.BuildpackID != "" && p.BuildpackPath == "" {
		if err := p.InferBuildpackPath(); err != nil {
			return err
		}
	}

	if p.BuildpackVersion == "" {
		if err := p.InferBuildpackVersion(); err != nil {
			return err
		}
	}

	if p.RegistryName == "" {
		p.RegistryName = p.BuildpackID
	}

	return p.Execute()
}
