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

package packager

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb/v2"
	"github.com/paketo-buildpacks/libpak/v2/carton"
	"github.com/paketo-buildpacks/libpak/v2/effect"
)

type BundleBuildpack struct {
	// BuildpackPath is the location to the buildpack source files
	BuildpackPath string

	// BuildpackID is the id of the buildpack you want to package
	BuildpackID string

	// Version is a version to substitute into an existing buildpack.toml.
	BuildpackVersion string

	// CacheLocation is the location to cache downloaded dependencies.
	CacheLocation string

	// DependencyFilters indicates which filters should be applied to exclude dependencies
	DependencyFilters []string

	// StrictDependencyFilters indicates that a filter must match both the ID and version, otherwise it must only match one of the two
	StrictDependencyFilters bool

	// IncludeDependencies indicates whether to include dependencies in build package.
	IncludeDependencies bool

	executor    effect.Executor
	exitHandler libcnb.ExitHandler
}

func NewBundleBuildpack() BundleBuildpack {
	return BundleBuildpack{
		executor: effect.NewExecutor(),
	}
}

func NewBundleBuildpackForTests(executor effect.Executor, exitHandler libcnb.ExitHandler) BundleBuildpack {
	return BundleBuildpack{
		executor:    executor,
		exitHandler: exitHandler,
	}
}

// InferBuildpackPath infers the buildpack path from the buildpack id
func (p *BundleBuildpack) InferBuildpackPath() error {
	root, found := os.LookupEnv("BP_ROOT")
	if !found {
		return fmt.Errorf("BP_ROOT must be set")
	}

	bpParts := strings.SplitN(p.BuildpackID, "/", 2)
	if len(bpParts) != 2 {
		return fmt.Errorf("invalid buildpack id: %s, must contain two parts that are `/` separated", p.BuildpackID)
	}

	bpType := bpParts[0]
	bpName := bpParts[1]

	switch bpType {
	case "paketobuildpacks":
		p.BuildpackPath = filepath.Join(root, "paketo-buildpacks", bpName)
	case "paketocommunity":
		p.BuildpackPath = filepath.Join(root, "paketo-community", bpName)
	default:
		p.BuildpackPath = filepath.Join(root, p.BuildpackID)
	}

	return nil
}

// InferBuildpackVersion from git state or default to DEV
func (p *BundleBuildpack) InferBuildpackVersion() error {
	buf := bytes.Buffer{}

	err := p.executor.Execute(effect.Execution{
		Command: "git",
		Args:    []string{"describe", "--tags", "--match", "v*"},
		Stdout:  &buf,
		Stderr:  io.Discard,
		Dir:     p.BuildpackPath,
	})
	if err != nil {
		return fmt.Errorf("unable to execute git command\n%w", err)
	}

	gitResult := strings.TrimSpace(buf.String())
	if gitResult == "" {
		gitResult = "DEV"
	}

	p.BuildpackVersion = strings.TrimPrefix(gitResult, "v")

	return nil
}

// CleanUpDockerImages removes dangling docker images created by the build process
func (p *BundleBuildpack) CleanUpDockerImages() error {
	buf := &bytes.Buffer{}
	err := p.executor.Execute(effect.Execution{
		Command: "docker",
		Args: []string{
			"image",
			"ls",
			"--quiet",
			"--no-trunc",
			"--filter",
			"dangling=true",
		},
		Stdout: buf,
		Stderr: io.Discard,
	})
	if err != nil {
		return fmt.Errorf("unable to execute `docker image ls` command\n%w", err)
	}

	imagesToClean := []string{}
	for _, img := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if img != "" {
			imagesToClean = append(imagesToClean, img)
		}
	}

	if len(imagesToClean) > 0 {
		err = p.executor.Execute(effect.Execution{
			Command: "docker",
			Args: append([]string{
				"image",
				"rm",
				"-f",
			}, imagesToClean...),
			Stdout: io.Discard,
			Stderr: io.Discard,
		})
		if err != nil {
			return fmt.Errorf("unable to execute `docker image rm` command\n%w", err)
		}
	}

	return nil
}

// ExecutePackage runs the package buildpack command
func (p *BundleBuildpack) ExecutePackage(tmpDir string) error {
	pullPolicy, found := os.LookupEnv("BP_PULL_POLICY")
	if !found {
		pullPolicy = "if-not-present"
	}

	err := p.executor.Execute(effect.Execution{
		Command: "pack",
		Args: []string{
			"buildpack",
			"package",
			p.BuildpackID,
			"--pull-policy", pullPolicy,
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Dir:    tmpDir,
	})
	if err != nil {
		return fmt.Errorf("unable to execute `pack buildpack package` command\n%w", err)
	}

	return nil
}

// CompilePackage compiles the buildpack's Go code
func (p *BundleBuildpack) CompilePackage(tmpDir string) {
	pkg := carton.Package{}
	pkg.Source = p.BuildpackPath
	pkg.Version = p.BuildpackVersion
	pkg.CacheLocation = p.CacheLocation
	pkg.DependencyFilters = p.DependencyFilters
	pkg.StrictDependencyFilters = p.StrictDependencyFilters
	pkg.IncludeDependencies = p.IncludeDependencies
	pkg.Destination = tmpDir

	options := []carton.Option{
		carton.WithExecutor(p.executor),
	}
	if p.exitHandler != nil {
		options = append(options, carton.WithExitHandler(p.exitHandler))
	}
	pkg.Create(options...)
}

// Execute runs the package buildpack command
func (p *BundleBuildpack) Execute() error {
	tmpDir, err := os.MkdirTemp("", "BundleBuildpack")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory\n%w", err)
	}

	// Compile the buildpack
	fmt.Println("➜ Compile Buildpack")
	p.CompilePackage(tmpDir)

	// package the buildpack
	fmt.Printf("➜ Package Buildpack: %s\n", p.BuildpackID)
	err = p.ExecutePackage(tmpDir)
	if err != nil {
		return fmt.Errorf("unable to package the buildpack\n%w", err)
	}

	// clean up
	fmt.Println("➜ Cleaning up Docker images")
	err = p.CleanUpDockerImages()
	if err != nil {
		return fmt.Errorf("unable to clean up docker images\n%w", err)
	}

	return nil
}
