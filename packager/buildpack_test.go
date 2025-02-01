/*
 * Copyright 2018-2024 the original author or authors.
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
package packager_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	exMocks "github.com/buildpacks/libcnb/v2/mocks"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/v2/effect"
	"github.com/paketo-buildpacks/libpak/v2/effect/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak-tools/packager"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it.Before(func() {
		t.Setenv("BP_ARCH", "amd64")
	})

	context("Infer Buildpack Path", func() {
		context("BP_ROOT is not set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("BP_ROOT")).ToNot(HaveOccurred())
			})

			it("errors if BP_ROOT isn't set", func() {
				p := packager.NewBundleBuildpack()
				Expect(p.InferBuildpackPath()).To(MatchError("BP_ROOT must be set"))
			})
		})

		context("BP_ROOT is set", func() {
			var p packager.BundleBuildpack

			it.Before(func() {
				t.Setenv("BP_ROOT", "/some/path")

				p = packager.NewBundleBuildpack()
			})

			it("errors if buildpack id is invalid", func() {
				p.BuildpackID = "invalid"
				Expect(p.InferBuildpackPath()).To(MatchError("invalid buildpack id: invalid, must contain two parts that are `/` separated"))
			})

			it("infers the buildpack path from the buildpack id `paketobuildpacks`", func() {
				p.BuildpackID = "paketobuildpacks/foo"
				Expect(p.InferBuildpackPath()).To(Succeed())
				Expect(p.BuildpackPath).To(Equal("/some/path/paketo-buildpacks/foo"))
			})

			it("infers the buildpack path from the buildpack id `paketocommunity`", func() {
				p.BuildpackID = "paketocommunity/foo"
				Expect(p.InferBuildpackPath()).To(Succeed())
				Expect(p.BuildpackPath).To(Equal("/some/path/paketo-community/foo"))
			})

			it("infers the buildpack path from the buildpack id `paketo-buildpacks`", func() {
				p.BuildpackID = "paketo-buildpacks/foo"
				Expect(p.InferBuildpackPath()).To(Succeed())
				Expect(p.BuildpackPath).To(Equal("/some/path/paketo-buildpacks/foo"))
			})
		})
	})

	context("Infer Buildpack Version", func() {
		var mockExecutor *mocks.Executor

		it.Before(func() {
			mockExecutor = &mocks.Executor{}
		})

		it("runs git tags to get the version", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "git" &&
					e.Args[0] == "describe" &&
					e.Args[1] == "--tags" &&
					e.Args[2] == "--match" &&
					e.Args[3] == "v*"
			})).Return(func(ex effect.Execution) error {
				_, err := ex.Stdout.Write([]byte("v1.2.3"))
				Expect(err).ToNot(HaveOccurred())
				return nil
			})

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.InferBuildpackVersion()).To(Succeed())
			Expect(p.BuildpackVersion).To(Equal("1.2.3"))
		})

		it("runs git tags to get the version with a hash", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "git" &&
					e.Args[0] == "describe" &&
					e.Args[1] == "--tags" &&
					e.Args[2] == "--match" &&
					e.Args[3] == "v*"
			})).Return(func(ex effect.Execution) error {
				_, err := ex.Stdout.Write([]byte("v1.42.0-14-g917ef34"))
				Expect(err).ToNot(HaveOccurred())
				return nil
			})

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.InferBuildpackVersion()).To(Succeed())
			Expect(p.BuildpackVersion).To(Equal("1.42.0-14-g917ef34"))
		})

		it("runs git tags to get the version but nothing is returned", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "git" &&
					e.Args[0] == "describe" &&
					e.Args[1] == "--tags" &&
					e.Args[2] == "--match" &&
					e.Args[3] == "v*"
			})).Return(func(ex effect.Execution) error {
				_, err := ex.Stdout.Write([]byte("  "))
				Expect(err).ToNot(HaveOccurred())
				return nil
			})

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.InferBuildpackVersion()).To(Succeed())
			Expect(p.BuildpackVersion).To(Equal("DEV"))
		})

		it("runs git tags which fails", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "git" &&
					e.Args[0] == "describe" &&
					e.Args[1] == "--tags" &&
					e.Args[2] == "--match" &&
					e.Args[3] == "v*"
			})).Return(fmt.Errorf("some-error"))

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.InferBuildpackVersion()).To(MatchError(ContainSubstring("some-error")))
		})
	})

	context("Clean up Docker images", func() {
		var mockExecutor *mocks.Executor

		it.Before(func() {
			mockExecutor = &mocks.Executor{}
		})

		it("fetching docker images to prune fails", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "docker" &&
					e.Args[0] == "image" &&
					e.Args[1] == "ls" &&
					e.Args[2] == "--quiet" &&
					e.Args[3] == "--no-trunc" &&
					e.Args[4] == "--filter" &&
					e.Args[5] == "dangling=true"
			})).Return(fmt.Errorf("docker fails"))

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.CleanUpDockerImages()).To(MatchError(ContainSubstring("docker fails")))
		})

		it("fetching docker images gives empty list", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "docker" &&
					e.Args[0] == "image" &&
					e.Args[1] == "ls" &&
					e.Args[2] == "--quiet" &&
					e.Args[3] == "--no-trunc" &&
					e.Args[4] == "--filter" &&
					e.Args[5] == "dangling=true"
			})).Return(func(ex effect.Execution) error {
				_, err := ex.Stdout.Write([]byte("  "))
				Expect(err).ToNot(HaveOccurred())
				return nil
			})

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.CleanUpDockerImages()).To(Succeed())
		})

		it("fetching docker images gives list of images", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "docker" &&
					e.Args[0] == "image" &&
					e.Args[1] == "ls" &&
					e.Args[2] == "--quiet" &&
					e.Args[3] == "--no-trunc" &&
					e.Args[4] == "--filter" &&
					e.Args[5] == "dangling=true"
			})).Return(func(ex effect.Execution) error {
				_, err := ex.Stdout.Write([]byte("sha256:31699bb7e3abe178dd1b3a494c64fc5efbf8488cb635253d25baaf6038e24b54\r\nsha256:4e916e5e31f3ac0d84c436ae60a8fa7c670b05051532f36193cd5fe504e2aee6\r\n"))
				Expect(err).ToNot(HaveOccurred())
				return nil
			})

			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "docker" &&
					e.Args[0] == "image" &&
					e.Args[1] == "rm" &&
					e.Args[2] == "-f" &&
					e.Args[3] == "sha256:31699bb7e3abe178dd1b3a494c64fc5efbf8488cb635253d25baaf6038e24b54" &&
					e.Args[4] == "sha256:4e916e5e31f3ac0d84c436ae60a8fa7c670b05051532f36193cd5fe504e2aee6"
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackPath = "/some/path"

			Expect(p.CleanUpDockerImages()).To(Succeed())
		})
	})

	context("Run pack buildpack package", func() {
		var mockExecutor *mocks.Executor

		it.Before(func() {
			mockExecutor = &mocks.Executor{}
		})

		it("defaults pull policy to if-not-present", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "pack" &&
					e.Args[0] == "buildpack" &&
					e.Args[1] == "package" &&
					e.Args[2] == "some-id" &&
					e.Args[3] == "--pull-policy" &&
					e.Args[4] == "if-not-present" &&
					e.Args[5] == "--target" &&
					e.Args[6] == "linux/amd64" &&
					e.Dir == "/some/path"
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackID = "some-id"

			Expect(p.ExecutePackage("/some/path")).To(Succeed())
		})

		it("include registry prefix if set", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "pack" &&
					e.Args[0] == "buildpack" &&
					e.Args[1] == "package" &&
					e.Args[2] == "docker.io/some-other-id/image" &&
					e.Args[3] == "--pull-policy" &&
					e.Args[4] == "if-not-present" &&
					e.Args[5] == "--publish" &&
					e.Dir == "/some/path"
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackID = "some-id"
			p.RegistryName = "docker.io/some-other-id/image"
			p.Publish = true

			Expect(p.ExecutePackage("/some/path")).To(Succeed())
		})

		context("BP_PULL_POLICY is set", func() {
			it.Before(func() {
				t.Setenv("BP_PULL_POLICY", "always")
			})

			it("uses a specific pull policy", func() {
				mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
					return e.Command == "pack" &&
						e.Args[0] == "buildpack" &&
						e.Args[1] == "package" &&
						e.Args[2] == "some-id" &&
						e.Args[3] == "--pull-policy" &&
						e.Args[4] == "always" &&
						e.Args[5] == "--target" &&
						e.Args[6] == "linux/amd64" &&
						e.Dir == "/some/path"
				})).Return(nil)

				p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
				p.BuildpackID = "some-id"

				Expect(p.ExecutePackage("/some/path")).To(Succeed())
			})
		})

		it("includes additional args", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "pack" &&
					e.Args[0] == "buildpack" &&
					e.Args[1] == "package" &&
					e.Args[2] == "some-id" &&
					e.Args[3] == "--pull-policy" &&
					e.Args[4] == "if-not-present" &&
					e.Args[5] == "--target" &&
					e.Args[6] == "linux/amd64" &&
					e.Args[7] == "--some-more-args" &&
					e.Dir == "/some/path"
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackID = "some-id"

			Expect(p.ExecutePackage("/some/path", "--some-more-args")).To(Succeed())
		})
	})

	context("CompilePackage", func() {
		var (
			mockExecutor    *mocks.Executor
			mockExitHandler exMocks.ExitHandler
		)

		it.Before(func() {
			mockExecutor = &mocks.Executor{}
			mockExitHandler = exMocks.ExitHandler{}

			t.Setenv("BP_ARCH", "amd64")
		})

		it("compiles the package", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "pack" &&
					e.Args[0] == "buildpack" &&
					e.Args[1] == "package" &&
					e.Args[2] == "some-id" &&
					e.Args[3] == "--pull-policy" &&
					e.Args[4] == "if-not-present" &&
					e.Args[5] == "--target" &&
					e.Args[6] == "linux/amd64" &&
					e.Dir == "/some/path"
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, &mockExitHandler)
			p.BuildpackID = "some-id"

			Expect(p.ExecutePackage("/some/path")).To(Succeed())
		})
	})

	context("Bundles a Composite", func() {
		var (
			buildpackPath string
			buildPath     string
			mockExecutor  *mocks.Executor
		)

		it.Before(func() {
			buildpackPath = t.TempDir()
			buildPath = t.TempDir()
			mockExecutor = &mocks.Executor{}

			Expect(os.WriteFile(filepath.Join(buildpackPath, "package.toml"), []byte("some-toml"), 0600)).To(Succeed())
		})

		it("inserts the URI to package.toml and runs pack buildpack package", func() {
			mockExecutor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				Expect(e.Command).To(Equal("pack"))
				Expect(e.Args).To(HaveExactElements([]string{
					"buildpack",
					"package",
					"some-id",
					"--pull-policy",
					"if-not-present",
					"--target",
					"linux/amd64",
					"--config",
					filepath.Join(buildPath, "/package.toml"),
					"--flatten",
				}))
				Expect(e.Dir).To(Equal(buildpackPath))
				return true
			})).Return(nil)

			p := packager.NewBundleBuildpackForTests(mockExecutor, nil)
			p.BuildpackID = "some-id"
			p.BuildpackPath = buildpackPath

			Expect(p.BundleComposite(buildPath)).To(Succeed())

			packageToml := filepath.Join(buildPath, "package.toml")
			Expect(packageToml).To(BeARegularFile())
			contents, err := os.ReadFile(packageToml)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(HavePrefix(fmt.Sprintf("[buildpack]\nuri = \"%s\"\n\n", buildpackPath)))
		})
	})
}
