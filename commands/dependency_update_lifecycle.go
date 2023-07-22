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

func DependencyUpdateLifecycleCommand() *cobra.Command {
	l := carton.LifecycleDependency{}

	var dependencyUpdateLifecycleCmd = &cobra.Command{
		Use:   "lifecycle",
		Short: "Update a lifecycle dependency",
		Run: func(cmd *cobra.Command, args []string) {
			if l.BuilderPath == "" {
				log.Fatal("builder-toml must be set")
			}

			if l.Version == "" {
				log.Fatal("version must be set")
			}

			l.Update()
		},
	}

	dependencyUpdateLifecycleCmd.Flags().StringVar(&l.BuilderPath, "builder-toml", "", "path to builder.toml")
	dependencyUpdateLifecycleCmd.Flags().StringVar(&l.Version, "version", "", "the new version of the dependency")

	return dependencyUpdateLifecycleCmd
}
