# Tools for Buildpacks

This repository pulls together and publishes a number of helpful tools for the management and release of buildpacks.

## `create-package``

The `create-package` tool creates a `libpak.Package` and calls `libpak.Package.Create()`. This takes a Paketo buildpack written in Go and packages is it into a buildpack. That involves compiling the source code, possibly copying in additional resource files, and generating the buildpack in the given output directory.

```
> create-package -h
Usage of Create Package:
  -cache-location string
    	path to cache downloaded dependencies (default: $PWD/dependencies)
  -dependency-filter value
    	one or more filters that are applied to exclude dependencies
  -destination string
    	path to the build package destination directory
  -include-dependencies
    	whether to include dependencies (default: false)
  -source string
    	path to build package source directory (default: $PWD) (default "/Users/dmikusa/Code/OSS/paketo-buildpacks/libpak-tools")
  -strict-filters
    	require filter to match all data or just some data (default: false)
  -version string
    	version to substitute into buildpack.toml/extension.toml
```

## `update-build-image-dependency`

The `update-build-image-dependency` tool is used to update depenencies in a build image dependency in a builder configuration file. It takes as an argument the builder configuration file and the new version.

```
> update-build-image-dependency -h
Usage of Update Build Image Dependency:
  -builder-toml string
    	path to builder.toml
  -version string
    	the new version of the dependency
```

## `update-package-dependency`

The `update-package-dependency` tool is used to update package dependencies, which are references to other buildpacks, in a builder definition (i.e. `builder.toml`), package definition (i.e. `package.toml`), or a buildpack definition (i.e. `buildpack.toml`, but only if it is a composite buildpack).

```
> update-package-dependency -h
Usage of Update Package Dependency:
  -builder-toml string
    	path to builder.toml
  -buildpack-toml string
    	path to buildpack.toml
  -id string
    	the id of the dependency
  -package-toml string
    	path to package.toml
  -version string
    	the new version of the dependency
```

## `update-buildmodule-dependency`

The `update-buildmodule-dependency` tool is used to update binary dependencies that are tracked in the build module metadata (i.e. `buildpack.toml` or `extension.toml` in the `[metadata.dependencies] block`). It allows you to update specific fields as the dependency changes.

```
> update-buildmodule-dependency -h
Usage of Update Build Module Dependency:
  -buildmodule-toml string
    	path to buildpack.toml or extension.toml
  -cpe string
    	the new version use in all CPEs, if not set defaults to version
  -cpe-pattern string
    	the cpe version pattern of the dependency, if not set defaults to version-pattern
  -id string
    	the id of the dependency
  -purl string
    	the new purl version of the dependency, if not set defaults to version
  -purl-pattern string
    	the purl version pattern of the dependency, if not set defaults to version-pattern
  -sha256 string
    	the new sha256 of the dependency
  -uri string
    	the new uri of the dependency
  -version string
    	the new version of the dependency
  -version-pattern string
    	the version pattern of the dependency
```

## `update-lifecycle-dependency`

The `update-lifecycle-dependency` tool is used to update the lifecycle dependency in a builder configuration (i.e. `builder.toml`).

```
> update-lifecycle-dependency -h
Usage of Update Lifecycle Dependency:
  -builder-toml string
    	path to builder.toml
  -version string
    	the new version of the dependency
```

## Notes

All of the binaries use Go's `flag` module to parse arguments which defaults to a single dash prefixing arguments (i.e. `-some-arg`), but also supports the more common double-dash format (i.e. `--some-arg`).

## License

This library is released under version 2.0 of the [Apache License][a].

[a]: https://www.apache.org/licenses/LICENSE-2.0