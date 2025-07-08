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

package carton_test

import (
	"os"
	"testing"

	"github.com/buildpacks/libcnb/v2/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	libpakTesting "github.com/paketo-buildpacks/libpak/v2/testing"

	"github.com/paketo-buildpacks/libpak-tools/carton"
)

func testBuildpackDependency(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		f, err := os.CreateTemp("", "carton-buildpack-dependency")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates dependency", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-1"
uri           = "test-uri-1"
sha256        = "test-sha256-1"
stacks        = [ "test-stack" ]
source        = "test-source-uri-1"
source-sha256 = "test-source-sha256-1"
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			Source:          "test-source-uri-2",
			SourceSHA256:    "test-source-sha256-2",
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
source        = "test-source-uri-2"
source-sha256 = "test-source-sha256-2"
`))
	})

	it("updates dependency with purl & cpes", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-1?arch=amd64"
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
`))
	})

	it("updates dependency with purls & cpes", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
	[buildpack]
	id = "some-buildpack"
	name = "Some Buildpack"
	version = "1.2.3"

	[[metadata.dependencies]]
	id      = "test-id"
	name    = "Test Name"
	version = "test-version-1"
	uri     = "test-uri-1"
	sha256  = "test-sha256-1"
	stacks  = [ "test-stack" ]
	purls    = ["pkg:generic/test-jre@different-version-1?arch=arm64"]
	cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
	`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "arm64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
	[buildpack]
	id = "some-buildpack"
	name = "Some Buildpack"
	version = "1.2.3"

	[[metadata.dependencies]]
	id      = "test-id"
	name    = "Test Name"
	version = "test-version-2"
	uri     = "test-uri-2"
	sha256  = "test-sha256-2"
	stacks  = [ "test-stack" ]
	purls    = ["pkg:generic/test-jre@different-version-2?arch=arm64"]
	cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
	`))
	})

	it("updates dependency with source & sourceSha", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"
[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-1?arch=amd64"
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
			Source:          "test-new-source",
			SourceSHA256:    "test-new-source-sha",
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-2"
uri           = "test-uri-2"
sha256        = "test-sha256-2"
stacks        = [ "test-stack" ]
purl          = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes          = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
source        = "test-new-source"
source-sha256 = "test-new-source-sha"
`))
	})

	it("updates dependency with checksum & source-checksum", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"
[[metadata.dependencies]]
id       = "test-id"
name     = "Test Name"
version  = "test-version-1"
uri      = "test-uri-1"
checksum = "sha256:test-sha256-1"
stacks   = [ "test-stack" ]
purls    = ["pkg:generic/test-jre@different-version-1?arch=amd64"]
cpes     = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
			Source:          "test-new-source",
			SourceSHA256:    "test-new-source-sha",
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id              = "test-id"
name            = "Test Name"
version         = "test-version-2"
uri             = "test-uri-2"
checksum        = "sha256:test-sha256-2"
stacks          = [ "test-stack" ]
purls           = ["pkg:generic/test-jre@different-version-2?arch=amd64"]
cpes            = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
source          = "test-new-source"
source-checksum = "sha256:test-new-source-sha"
`))
	})

	it("updates multiple dependencies with different versions", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-1"
uri           = "test-uri-1"
sha256        = "test-sha256-1"
stacks        = [ "test-stack" ]
purl          = "pkg:generic/test-jre@different-version-1?arch=amd64"
cpes          = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
source        = "test-source-uri-1"
source-sha256 = "test-source-sha256-1"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-2"
uri           = "test-uri-2"
sha256        = "test-sha256-2"
stacks        = [ "test-stack" ]
purl          = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes          = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
source        = "test-source-uri-2"
source-sha256 = "test-source-sha256-2"
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-3",
			URI:             "test-uri-3",
			Version:         "test-version-3",
			VersionPattern:  `test-version-1`,
			PURL:            "different-version-3",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-3:patch3",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
			Source:          "test-source-uri-3",
			SourceSHA256:    "test-source-sha256-3",
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-3"
uri           = "test-uri-3"
sha256        = "test-sha256-3"
stacks        = [ "test-stack" ]
purl          = "pkg:generic/test-jre@different-version-3?arch=amd64"
cpes          = ["cpe:2.3:a:test-vendor:test-product:test-version-3:patch3:*:*:*:*:*:*:*"]
source        = "test-source-uri-3"
source-sha256 = "test-source-sha256-3"

[[metadata.dependencies]]
id            = "test-id"
name          = "Test Name"
version       = "test-version-2"
uri           = "test-uri-2"
sha256        = "test-sha256-2"
stacks        = [ "test-stack" ]
purl          = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes          = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
source        = "test-source-uri-2"
source-sha256 = "test-source-sha256-2"
`))
	})

	it("updates dependency with missing purl, still updates cpe", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
`))
	})

	it("updates dependency with invalid purl, still updates cpe", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
purl    = 1234
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
purl    = 1234
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
`))
	})

	it("updates dependency with invalid cpe, still updates purl", func() {
		Expect(os.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-1?arch=amd64"
cpes    = 1234
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
			PURL:            "different-version-2",
			PURLPattern:     `different-version-[\d]`,
			CPE:             "test-version-2:patch2",
			CPEPattern:      `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(libpakTesting.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes    = 1234
`))
	})

	it("updates indented dependency", func() {
		Expect(os.WriteFile(path, []byte(`# it should preserve
#   these comments
#      exactly

api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
  id      = "test-id"
  name    = "Test Name"
  version = "test-version-1"
  uri     = "test-uri-1"
  sha256  = "test-sha256-1"
  stacks  = [ "test-stack" ]
`), 0600)).To(Succeed())

		d := carton.BuildModuleDependency{
			BuildModulePath: path,
			ID:              "test-id",
			Arch:            "amd64",
			SHA256:          "test-sha256-2",
			URI:             "test-uri-2",
			Version:         "test-version-2",
			VersionPattern:  `test-version-[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		body, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(HavePrefix(`# it should preserve
#   these comments
#      exactly

api = "0.6"`))
		Expect(body).To(libpakTesting.MatchTOML(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
  id      = "test-id"
  name    = "Test Name"
  version = "test-version-2"
  uri     = "test-uri-2"
  sha256  = "test-sha256-2"
  stacks  = [ "test-stack" ]
`))
	})
}
