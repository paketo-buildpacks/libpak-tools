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

	"github.com/paketo-buildpacks/libpak/v2/carton"
	"github.com/spf13/cobra"
)

func DependencyUpdatePackageCommand() *cobra.Command {
	p := carton.PackageDependency{}

	var dependencyUpdatePackageCmd = &cobra.Command{
		Use:   "package",
		Short: "Update a package dependency",
		Run: func(cmd *cobra.Command, args []string) {
			if p.BuilderPath == "" && p.BuildpackPath == "" && p.PackagePath == "" {
				log.Fatal("builder-toml, buildpack-toml, or package-toml must be set")
			}

			if p.ID == "" {
				log.Fatal("id must be set")
			}

			if p.Version == "" {
				log.Fatal("version must be set")
			}

			p.Update()
		},
	}

	dependencyUpdatePackageCmd.Flags().StringVar(&p.BuilderPath, "builder-toml", "", "path to builder.toml")
	dependencyUpdatePackageCmd.Flags().StringVar(&p.BuildpackPath, "buildpack-toml", "", "path to buildpack.toml")
	dependencyUpdatePackageCmd.Flags().StringVar(&p.ID, "id", "", "the id of the dependency")
	dependencyUpdatePackageCmd.Flags().StringVar(&p.PackagePath, "package-toml", "", "path to package.toml")
	dependencyUpdatePackageCmd.Flags().StringVar(&p.Version, "version", "", "the new version of the dependency")

	return dependencyUpdatePackageCmd
}
