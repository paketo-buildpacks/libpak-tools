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
	"fmt"
	"log"
	"os"

	"github.com/paketo-buildpacks/libpak/v2/carton"
	"github.com/spf13/cobra"
)

func PackageCreateCommand() *cobra.Command {
	p := carton.Package{}

	var packageCreateCommand = &cobra.Command{
		Use:   "create",
		Short: "Create a package from buildpack source code",
		Run: func(cmd *cobra.Command, args []string) {
			if p.Destination == "" {
				log.Fatal("destination must be set")
			}

			p.Create()
		},
	}

	packageCreateCommand.Flags().StringVar(&p.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	packageCreateCommand.Flags().StringVar(&p.Destination, "destination", "", "path to the build package destination directory")
	packageCreateCommand.Flags().BoolVar(&p.IncludeDependencies, "include-dependencies", false, "whether to include dependencies (default: false)")
	packageCreateCommand.Flags().StringArrayVar(&p.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies")
	packageCreateCommand.Flags().BoolVar(&p.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data (default: false)")
	packageCreateCommand.Flags().StringVar(&p.Source, "source", defaultSource(), "path to build package source directory (default: $PWD)")
	packageCreateCommand.Flags().StringVar(&p.Version, "version", "", "version to substitute into buildpack.toml/extension.toml")

	return packageCreateCommand
}

func defaultSource() string {
	s, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("unable to get working directory\n%w", err))
	}

	return s
}
