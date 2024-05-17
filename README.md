# Tools for Buildpacks

This repository pulls together and publishes a number of helpful tools for the management and release of buildpacks.

## `libpak-tools package compile`

The `package compile` command creates a `libpak.Package` and calls `libpak.Package.Create()`. This takes a Paketo buildpack written in Go and packages is it into a buildpack. That involves compiling the source code, possibly copying in additional resource files, and generating the buildpack in the given output directory. The key is that the output of this command is a *directory*. If you want it to output an image, use `libpak-tools package bundle`.

```
> libpak-tools package compile -h
Compile buildpack source code

Usage:
  libpak-tools package compile [flags]

Flags:
      --cache-location string           path to cache downloaded dependencies (default: $PWD/dependencies)
      --dependency-filter stringArray   one or more filters that are applied to exclude dependencies
      --destination string              path to the build package destination directory
  -h, --help                            help for compile
      --include-dependencies            whether to include dependencies (default: false)
      --source string                   path to build package source directory (default: $PWD) (default "/Users/dmikusa/Code/OSS/paketo-buildpacks/libpak-tools")
      --strict-filters                  require filter to match all data or just some data (default: false)
      --version string                  version to substitute into buildpack.toml/extension.toml
```

## `libpak-tools package bundle`

The `package bundle` does the same thing as `libpak-tools package compile` but then runs `pack buildpack package` as well, so the output is a buildpack image.

```
> libpak-tools package bundle -h
Compile and package a single buildpack

Usage:
  libpak-tools package bundle [flags]

Flags:
      --buildpack-id string             id of the buildpack to use
      --buildpack-path string           path to buildpack directory
      --cache-location string           path to cache downloaded dependencies (default: $PWD/dependencies)
      --dependency-filter stringArray   one or more filters that are applied to exclude dependencies
  -h, --help                            help for bundle
      --include-dependencies            whether to include dependencies (default: false)
      --strict-filters                  require filter to match all data or just some data (default: false)
      --version string                  version to substitute into buildpack.toml/extension.toml
```

## `libpak-tools dependency update build-image`

The `dependency update build-image` command is used to update dependencies in a build image dependency in a builder configuration file. It takes as an argument the builder configuration file and the new version.

```
> libpak-tools dependency update build-image -h
Update a build image dependency

Usage:
  libpak-tools dependency update build-image [flags]

Flags:
      --builder-toml string   path to builder.toml
  -h, --help                  help for build-image
      --version string        the new version of the dependency
```

## `libpak-tools dependency update package`

The `dependency update package` command is used to update package dependencies, which are references to other buildpacks, in a builder definition (i.e. `builder.toml`), package definition (i.e. `package.toml`), or a buildpack definition (i.e. `buildpack.toml`, but only if it is a composite buildpack).

```
> libpak-tools dependency update package -h
Update a package dependency

Usage:
  libpak-tools dependency update package [flags]

Flags:
      --builder-toml string     path to builder.toml
      --buildpack-toml string   path to buildpack.toml
  -h, --help                    help for package
      --id string               the id of the dependency
      --package-toml string     path to package.toml
      --version string          the new version of the dependency
```

## `libpak-tools dependency update build-module`

The `dependency update build-module` command is used to update binary dependencies that are tracked in the build module metadata (i.e. `buildpack.toml` or `extension.toml` in the `[metadata.dependencies] block`). It allows you to update specific fields as the dependency changes.

```
> libpak-tools dependency update build-module -h
Update a build module dependency

Usage:
  libpak-tools dependency update build-module [flags]

Flags:
      --buildmodule-toml string   path to buildpack.toml or extension.toml
      --cpe string                the new version use in all CPEs, if not set defaults to version
      --cpe-pattern string        the cpe version pattern of the dependency, if not set defaults to version-pattern
  -h, --help                      help for build-module
      --id string                 the id of the dependency
      --purl string               the new purl version of the dependency, if not set defaults to version
      --purl-pattern string       the purl version pattern of the dependency, if not set defaults to version-pattern
      --sha256 string             the new sha256 of the dependency
      --uri string                the new uri of the dependency
      --version string            the new version of the dependency
      --version-pattern string    the version pattern of the dependency
```

## `libpak-tools dependency update lifecycle`

The `dependency update lifecycle` command is used to update the lifecycle dependency in a builder configuration (i.e. `builder.toml`).

```
> libpak-tools dependency update lifecycle -h
Update a lifecycle dependency

Usage:
  libpak-tools dependency update lifecycle [flags]

Flags:
      --builder-toml string   path to builder.toml
  -h, --help                  help for lifecycle
      --version string        the new version of the dependency
```

## License

This library is released under version 2.0 of the [Apache License][a].

[a]: https://www.apache.org/licenses/LICENSE-2.0
