/*
 * Copyright 2018-2023 the original author or authors.
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
	"log"

	"github.com/paketo-buildpacks/libpak/carton"
	"github.com/spf13/cobra"
)

func DependencyUpdateBuildModuleCommand() *cobra.Command {
	b := carton.BuildModuleDependency{}

	var dependencyUpdateBuildModuleCmd = &cobra.Command{
		Use:   "build-module",
		Short: "Update a build module dependency",
		Run: func(cmd *cobra.Command, args []string) {
			if b.BuildModulePath == "" {
				log.Fatal("buildmodule toml path must be set")
			}

			if b.ID == "" {
				log.Fatal("id must be set")
			}

			if b.SHA256 == "" {
				log.Fatal("sha256 must be set")
			}

			if b.URI == "" {
				log.Fatal("uri must be set")
			}

			if b.Version == "" {
				log.Fatal("version must be set")
			}

			if b.VersionPattern == "" {
				log.Fatal("version-pattern must be set")
			}

			if b.PURL == "" {
				b.PURL = b.Version
			}

			if b.PURLPattern == "" {
				b.PURLPattern = b.VersionPattern
			}

			if b.CPE == "" {
				b.CPE = b.Version
			}

			if b.CPEPattern == "" {
				b.CPEPattern = b.VersionPattern
			}

			b.Update()
		},
	}

	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.BuildModulePath, "buildmodule-toml", "", "path to buildpack.toml or extension.toml")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.ID, "id", "", "the id of the dependency")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.SHA256, "sha256", "", "the new sha256 of the dependency")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.URI, "uri", "", "the new uri of the dependency")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.Version, "version", "", "the new version of the dependency")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.VersionPattern, "version-pattern", "", "the version pattern of the dependency")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.PURL, "purl", "", "the new purl version of the dependency, if not set defaults to version")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.PURLPattern, "purl-pattern", "", "the purl version pattern of the dependency, if not set defaults to version-pattern")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.CPE, "cpe", "", "the new version use in all CPEs, if not set defaults to version")
	dependencyUpdateBuildModuleCmd.Flags().StringVar(&b.CPEPattern, "cpe-pattern", "", "the cpe version pattern of the dependency, if not set defaults to version-pattern")

	return dependencyUpdateBuildModuleCmd
}
