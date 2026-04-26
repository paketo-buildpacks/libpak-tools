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
	"runtime"
	"strings"

	"github.com/buildpacks/libcnb/v2"
	"github.com/paketo-buildpacks/libpak/v2/crush"
	"github.com/paketo-buildpacks/libpak/v2/effect"
	"github.com/paketo-buildpacks/libpak/v2/sherpa"

	"github.com/paketo-buildpacks/libpak-tools/carton"
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

	// RegistryName is the prefix to use when publishing the buildpack
	RegistryName string

	// Format for the output, defaults to image. Other option is file.
	Format string

	// OutputName is the name of the output file when format is set to file, defaults to buildpackage.cnb in the buildpack directory
	OutputName string

	// Publish indicates whether to publish the buildpack to the registry
	Publish bool

	// SkipClean will not clean up resources left over from the build process
	SkipClean bool

	executor    effect.Executor
	exitHandler libcnb.ExitHandler
}

func NewBundleBuildpack() BundleBuildpack {
	return BundleBuildpack{
		Format:     "image",
		OutputName: "buildpackage.cnb",
		executor:   effect.NewExecutor(),
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
		if strings.TrimSpace(img) != "" {
			imagesToClean = append(imagesToClean, strings.TrimSpace(img))
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
			return fmt.Errorf("unable to execute `docker image rm` command on images %v\n%w", imagesToClean, err)
		}
	}

	return nil
}

// ExecutePackage runs the package buildpack command
func (p *BundleBuildpack) ExecutePackage(workingDirectory string, additionalArgs ...string) error {
	pullPolicy, found := os.LookupEnv("BP_PULL_POLICY")
	if !found {
		pullPolicy = "if-not-present"
	}

	outputName := p.BuildpackID
	if p.RegistryName != "" {
		outputName = p.RegistryName
	}

	if p.Format == "file" {
		absBuildpackPath, err := filepath.Abs(p.BuildpackPath)
		if err != nil {
			return fmt.Errorf("unable to resolve absolute buildpack path: %w", err)
		}

		outputName = filepath.Join(absBuildpackPath, p.OutputName)
	}

	args := []string{
		"buildpack",
		"package",
		outputName,
		"--pull-policy", pullPolicy,
	}

	if p.Format == "file" && p.Publish {
		return fmt.Errorf("unable to package, cannot publish when format is set to file")
	} else if p.Format == "file" {
		args = append(args, "--format", "file")
	}

	if p.Publish {
		args = append(args, "--publish")
	} else {
		args = append(args, "--target", archFromSystem())
	}

	args = append(args, additionalArgs...)
	err := p.executor.Execute(effect.Execution{
		Command: "pack",
		Args:    args,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Dir:     workingDirectory,
	})
	if err != nil {
		return fmt.Errorf("unable to execute `pack buildpack package` command\n%w", err)
	}

	return nil
}

// CompilePackage compiles the buildpack's Go code
func (p *BundleBuildpack) CompilePackage(destDir string) {
	pkg := carton.Package{}
	pkg.Source = p.BuildpackPath
	pkg.Version = p.BuildpackVersion
	pkg.CacheLocation = p.CacheLocation
	pkg.DependencyFilters = p.DependencyFilters
	pkg.StrictDependencyFilters = p.StrictDependencyFilters
	pkg.IncludeDependencies = p.IncludeDependencies
	pkg.Destination = destDir

	options := []carton.Option{
		carton.WithExecutor(p.executor),
	}
	if p.exitHandler != nil {
		options = append(options, carton.WithExitHandler(p.exitHandler))
	}
	pkg.Create(options...)
}

// GZipBuildpack zips the buildpack source code and places it in the destination directory
func (p *BundleBuildpack) GZipBuildpack(destDir string) error {
	archiveFile, err := os.CreateTemp("", "buildpack-*.tgz")
	if err != nil {
		return fmt.Errorf("unable to create temporary archive file\n%w", err)
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)

	err = crush.CreateTarGz(archiveFile, destDir)
	if err != nil {
		return fmt.Errorf("unable to create tar.gz archive\n%w", err)
	}

	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("unable to close archive file\n%w", err)
	}

	// move the archive to the buildpack directory, so it survives the temp build directory
	destArchivePath := filepath.Join(p.BuildpackPath, strings.Replace(p.OutputName, ".cnb", ".tgz", 1))
	if err := sherpa.CopyFileFrom(archivePath, destArchivePath); err != nil {
		return fmt.Errorf("unable to copy archive to buildpack directory\n%w", err)
	}

	return os.Remove(archivePath)
}

func (p *BundleBuildpack) CompileAndBundleComponent(buildDirectory string) error {
	// Compile the buildpack
	p.CompilePackage(buildDirectory)
	fmt.Println()

	// Make a gzip of the buildpack binaries
	err := p.GZipBuildpack(buildDirectory)
	if err != nil {
		return fmt.Errorf("unable to gzip buildpack source\n%w", err)
	}

	// package the buildpack
	return p.ExecutePackage(buildDirectory)
}

func (p *BundleBuildpack) BundleComposite(buildDirectory string) error {
	// Make a modified package.toml in the temp directory
	packageTomlPath, err := copyPackageTomlAndAddURI(p.BuildpackPath, buildDirectory)
	if err != nil {
		return fmt.Errorf("unable to copy package.toml and add URI\n%w", err)
	}

	// prepare extra arguments
	args := []string{
		"--config", packageTomlPath,
	}

	if !sherpa.ResolveBool("BP_FLATTEN_DISABLED") {
		args = append(args, "--flatten")
	}

	err = p.GZipBuildpack(buildDirectory)
	if err != nil {
		return fmt.Errorf("unable to gzip buildpack source\n%w", err)
	}

	// we still package from the buildpack directory though, only the package.toml is in the temp directory
	fmt.Printf("➜ Package Buildpack: %s\n", p.BuildpackID)
	return p.ExecutePackage(p.BuildpackPath, args...)
}

func copyPackageTomlAndAddURI(buildpackPath, destDir string) (string, error) {
	if err := sherpa.CopyFileFrom(filepath.Join(buildpackPath, "buildpack.toml"), filepath.Join(destDir, "buildpack.toml")); err != nil {
		return "", fmt.Errorf("unable to copy buildpack.toml\n%w", err)
	}

	inputPackageToml, err := os.Open(filepath.Join(buildpackPath, "package.toml"))
	if err != nil {
		return "", fmt.Errorf("unable to open package.toml\n%w", err)
	}
	defer inputPackageToml.Close()

	outputPackageTomlPath := filepath.Join(destDir, "package.toml")
	outputPackageToml, err := os.Create(outputPackageTomlPath)
	if err != nil {
		return "", fmt.Errorf("unable to create package.toml\n%w", err)
	}
	defer outputPackageToml.Close()

	_, err = outputPackageToml.WriteString(fmt.Sprintf("[buildpack]\nuri = \"%s\"\n\n", destDir))
	if err != nil {
		return "", fmt.Errorf("unable to write uri\n%w", err)
	}

	_, err = io.Copy(outputPackageToml, inputPackageToml)
	if err != nil {
		return "", fmt.Errorf("unable to copy rest of package.toml\n%w", err)
	}

	return outputPackageTomlPath, nil
}

// Execute runs the package buildpack command
func (p *BundleBuildpack) Execute() error {
	buildDirectory, err := os.MkdirTemp("", "BundleBuildpack")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory\n%w", err)
	}

	// we use existence of main.go to determine if we are packaging a component or composite buildpack
	mainCmdPath := filepath.Join(p.BuildpackPath, "cmd/main/main.go")
	if componentBp, err := sherpa.FileExists(mainCmdPath); err != nil {
		return fmt.Errorf("unable to check if file exists\n%w", err)
	} else if componentBp {
		p.CompileAndBundleComponent(buildDirectory)
	} else {
		p.BundleComposite(buildDirectory)
	}

	// clean up
	if !p.SkipClean {
		fmt.Println("➜ Cleaning up Docker images")
		err = p.CleanUpDockerImages()
		if err != nil {
			return fmt.Errorf("unable to clean up docker images\n%w", err)
		}
	}

	return nil
}

func archFromSystem() string {
	archFromEnv, ok := os.LookupEnv("BP_ARCH")
	if !ok {
		archFromEnv = runtime.GOARCH
	}

	return "linux/" + archFromEnv
}
