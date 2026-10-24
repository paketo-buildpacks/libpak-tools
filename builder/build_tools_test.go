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
package builder_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak-tools/builder"
)

func testBuildTools(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("updates buildpack details for a tool", func() {
		it("updates keys in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"buildpack": map[string]interface{}{
					"id":          "id",
					"name":        "name",
					"description": "desc",
					"homepage":    "homepage",
					"version":     "version",
					"foo":         "bar",
				},
			}

			builder.UpdateBuildpackDetailsForTool(builder.BuildTool{
				BuildpackID: "paketo-buildpacks/gradle",
				Description: "new-desc",
				Homepage:    "new-homepage",
				Name:        "new-name",
				ToolID:      "gradle",
			}, "new-version")(buildpackTOML)

			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("description", "new-desc"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("homepage", "new-homepage"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("id", "paketo-buildpacks/gradle"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("name", "new-name"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("version", "new-version"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("foo", "bar"))
		})
	})

	context("filters buildpack dependencies", func() {
		it("keeps only dependencies for the selected tool", func() {
			buildpackTOML := map[string]interface{}{
				"metadata": map[string]interface{}{
					"dependencies": []map[string]interface{}{
						{"id": "lein", "version": "1.0.0"},
						{"id": "clojure", "version": "1.0.0"},
						{"id": "gradle", "version": "1.0.0"},
						{"id": "maven", "version": "1.0.0"},
						{"id": "mvnd", "version": "1.0.0"},
						{"id": "sbt", "version": "1.0.0"},
					},
					"dependency-constraints": []map[string]interface{}{
						{"id": "lein", "constraint": "*"},
						{"id": "clojure", "constraint": "*"},
						{"id": "gradle", "constraint": "9.*"},
						{"id": "maven", "constraint": "3.*"},
						{"id": "mvnd", "constraint": "1.*"},
						{"id": "sbt", "constraint": "1.*"},
					},
				},
			}

			builder.KeepDependencies([]string{"maven", "mvnd"})(buildpackTOML)

			dependencies := buildpackTOML["metadata"].(map[string]interface{})["dependencies"].([]map[string]interface{})
			Expect(dependencies).To(HaveLen(2))
			Expect(dependencies[0]).To(HaveKeyWithValue("id", "maven"))
			Expect(dependencies[1]).To(HaveKeyWithValue("id", "mvnd"))

			constraints := buildpackTOML["metadata"].(map[string]interface{})["dependency-constraints"].([]map[string]interface{})
			Expect(constraints).To(HaveLen(2))
			Expect(constraints[0]).To(HaveKeyWithValue("id", "maven"))
			Expect(constraints[1]).To(HaveKeyWithValue("id", "mvnd"))
		})

		it("leaves the metadata alone when there are no dependencies", func() {
			buildpackTOML := map[string]interface{}{
				"metadata": map[string]interface{}{},
			}

			Expect(func() { builder.KeepDependencies([]string{"gradle"})(buildpackTOML) }).NotTo(Panic())
			Expect(buildpackTOML["metadata"]).To(Equal(map[string]interface{}{}))
		})
	})
}
