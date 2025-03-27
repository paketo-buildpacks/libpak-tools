/*
 * Copyright 2018-2025 the original author or authors.
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

func testBuilder(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("updates buildpack.toml data", func() {
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

			builder.UpdateBuildpackDetails(builder.JVMVendor{"new-desc", "new-homepage", "new-id", "new-name"}, "new-version")(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("buildpack"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("description", "new-desc"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("homepage", "new-homepage"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("id", "new-id"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("name", "new-name"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("version", "new-version"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("foo", "bar"))
		})

		it("adds keys that dont exist in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"buildpack": map[string]interface{}{
					"id":   "id",
					"name": "name",
					"foo":  "bar",
				},
			}

			builder.UpdateBuildpackDetails(builder.JVMVendor{"new-desc", "new-homepage", "new-id", "new-name"}, "new-version")(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("buildpack"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("description", "new-desc"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("homepage", "new-homepage"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("id", "new-id"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("name", "new-name"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("version", "new-version"))
			Expect(buildpackTOML["buildpack"]).To(HaveKeyWithValue("foo", "bar"))
		})
	})

	context("updates buildpack.toml configurations", func() {
		it("updates keys in buildpack.toml", func() {
			buildpackTOML := map[string]interface{}{
				"metadata": map[string]interface{}{
					"configurations": []map[string]interface{}{
						{
							"name":    "foo",
							"default": "test-default",
						},
						{
							"name":    "baz",
							"default": "other-default",
						},
					},
				},
			}

			builder.UpdateBuildpackConfiguration(map[string]interface{}{
				"foo": "update-value",
			})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("configurations"))

			cfgList := buildpackTOML["metadata"].(map[string]interface{})["configurations"].([]map[string]interface{})
			Expect(cfgList).To(HaveLen(2))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("name", "foo")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("default", "update-value")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("name", "baz")))
			Expect(cfgList).To(ContainElement(HaveKeyWithValue("default", "other-default")))
		})
	})

	context("metadata has a list of dependencies", func() {
		var buildpackTOML map[string]interface{}

		it.Before(func() {
			buildpackTOML = map[string]interface{}{
				"metadata": map[string]interface{}{
					"dependencies": []map[string]interface{}{
						{
							"id": "jdk-foo",
						},
						{
							"id": "jre-foo",
						},
						{
							"id": "jre-bar",
						},
						{
							"id": "jdk-baz",
						},
					},
				},
			}
		})

		it("removes everything", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(0))
		})

		it("removes dependencies not in the vendor list", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{"foo"})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(2))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(
				ContainElements(HaveKeyWithValue("id", "jdk-foo"), HaveKeyWithValue("id", "jre-foo")))
		})

		it("removes dependencies not in the vendor list", func() {
			builder.RemoveDependenciesUnlessInVendorList([]string{"baz"})(buildpackTOML)

			Expect(buildpackTOML).To(HaveKey("metadata"))
			Expect(buildpackTOML["metadata"]).To(HaveKey("dependencies"))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(HaveLen(1))
			Expect(buildpackTOML["metadata"].(map[string]interface{})["dependencies"]).To(
				ContainElements(HaveKeyWithValue("id", "jdk-baz")))
		})
	})
}
