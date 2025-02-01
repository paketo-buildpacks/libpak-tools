/*
 * Copyright 2018-2020 the original author or authors.
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

package carton

import (
	"fmt"
	"os"
	"strings"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/utils"

	"github.com/paketo-buildpacks/libpak-tools/internal"
)

type PackageDependency struct {
	BuilderPath   string
	BuildpackPath string
	ID            string
	Version       string
	PackagePath   string
}

func (p PackageDependency) Update(options ...Option) {
	config := Config{
		exitHandler: utils.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := log.NewPaketoLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", log.FormatIdentity(p.ID, p.Version))

	if p.BuilderPath != "" {
		if err := internal.UpdateTOMLFile(p.BuilderPath, updateByKey("buildpacks", p.ID, p.Version)); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to update %s\n%w", p.BuilderPath, err))
		}
	}

	if p.PackagePath != "" {
		if err := internal.UpdateTOMLFile(p.PackagePath, updateByKey("dependencies", p.ID, p.Version)); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to update %s\n%w", p.PackagePath, err))
		}
	}

	// Do we have a buildpack.toml with an order element? (composite buildpack)
	if p.BuildpackPath != "" {
		if err := internal.UpdateTOMLFile(p.BuildpackPath, func(md map[string]interface{}) {
			parts := strings.Split(p.ID, "/")
			id := strings.Join(parts[len(parts)-2:], "/")

			groupsUnwrapped, found := md["order"]
			if !found {
				return
			}

			groups, ok := groupsUnwrapped.([]map[string]interface{})
			if !ok {
				return
			}

			for _, group := range groups {
				buildpacksUnwrapped, found := group["group"]
				if !found {
					continue
				}

				buildpacks, ok := buildpacksUnwrapped.([]interface{})
				if !ok {
					continue
				}

				for _, bpw := range buildpacks {
					bp, ok := bpw.(map[string]interface{})
					if !ok {
						continue
					}

					bpIDUnwrappd, found := bp["id"]
					if !found {
						continue
					}

					bpID, ok := bpIDUnwrappd.(string)
					if !ok {
						continue
					}

					if bpID == id {
						bp["version"] = p.Version
					}
				}
			}
		}); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to update %s\n%w", p.BuildpackPath, err))
		}
	}
}

func updateByKey(key, id, version string) func(md map[string]interface{}) {
	return func(md map[string]interface{}) {
		valuesUnwrapped, found := md[key]
		if !found {
			return
		}

		values, ok := valuesUnwrapped.([]interface{})
		if !ok {
			return
		}

		for _, bpw := range values {
			bp, ok := bpw.(map[string]interface{})
			if !ok {
				continue
			}

			uriUnwrapped, found := bp["uri"]
			if !found {
				continue
			}

			uri, ok := uriUnwrapped.(string)
			if !ok {
				continue
			}

			if strings.HasPrefix(uri, fmt.Sprintf("docker://%s", id)) {
				parts := strings.Split(uri, ":")
				bp["uri"] = fmt.Sprintf("%s:%s", strings.Join(parts[0:2], ":"), version)
			}
		}
	}
}
