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

func DependencyUpdateBuildImageCommand() *cobra.Command {
	i := carton.BuildImageDependency{}

	var dependencyUpdateBuildImageCmd = &cobra.Command{
		Use:   "build-image",
		Short: "Update a build image dependency",
		Run: func(cmd *cobra.Command, args []string) {
			if i.BuilderPath == "" {
				log.Fatal("builder-toml must be set")
			}

			if i.Version == "" {
				log.Fatal("version must be set")
			}

			i.Update()
		},
	}

	dependencyUpdateBuildImageCmd.Flags().StringVar(&i.BuilderPath, "builder-toml", "", "path to builder.toml")
	dependencyUpdateBuildImageCmd.Flags().StringVar(&i.Version, "version", "", "the new version of the dependency")

	return dependencyUpdateBuildImageCmd
}
