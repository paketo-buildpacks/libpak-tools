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
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/paketo-buildpacks/libpak-tools/ci"
)

// CICalculateSemverBumpCommand returns the calculate-semver-bump subcommand.
func CICalculateSemverBumpCommand() *cobra.Command {
	var beforePath, afterPath string

	var calculateSemverBumpCmd = &cobra.Command{
		Use:   "calculate-semver-bump",
		Short: "Calculate the highest semver bump between two buildpack.toml files",
		RunE: func(cmd *cobra.Command, args []string) error {
			bump, err := ci.CalculateSemverBump(beforePath, afterPath)
			if err != nil {
				return err
			}

			fmt.Println(bump)
			return nil
		},
	}

	calculateSemverBumpCmd.Flags().StringVar(&beforePath, "before", "", "path to the buildpack.toml before updates")
	calculateSemverBumpCmd.Flags().StringVar(&afterPath, "after", "", "path to the buildpack.toml after updates")

	if err := calculateSemverBumpCmd.MarkFlagRequired("before"); err != nil {
		log.Fatal(err)
	}

	if err := calculateSemverBumpCmd.MarkFlagRequired("after"); err != nil {
		log.Fatal(err)
	}

	return calculateSemverBumpCmd
}
