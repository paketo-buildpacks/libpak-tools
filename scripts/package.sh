#!/usr/bin/env bash

set -eu
set -o pipefail

readonly ROOT_DIR="$(cd "$(dirname "${0}")/.." && pwd)"
readonly BUILD_DIR="${ROOT_DIR}/build"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${ROOT_DIR}/scripts/.util/print.sh"

function main {
  local version
  local platforms=()

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --version|-v)
        version="${2}"
        shift 2
        ;;

      --platform|-p)
        platforms+=("${2}")
        shift 2
        ;;

      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  if [[ -z "${version:-}" ]]; then
    usage
    echo
    util::print::error "--version is required"
  fi

  if [[ "${#platforms[@]}" -eq 0 ]]; then
    platforms=(
      "darwin/amd64"
      "darwin/arm64"
      "linux/386"
      "linux/amd64"
      "linux/arm64"
      "windows/386"
      "windows/amd64"
      "windows/arm64"
    )
  fi

  validate::platforms "${platforms[@]}"
  repo::prepare
  build::archives "${version}" "${platforms[@]}"
  checksums::write "${version}"
}

function usage() {
  cat <<-USAGE
package.sh --version <version> [OPTIONS]

Builds the libpak-tools CLI and packages the binaries into release archives.

OPTIONS
  --help                  -h                  prints the command usage
  --version <version>     -v <version>        specifies the version number to embed in the binaries
  --platform <os>/<arch>  -p <os>/<arch>      build only this GOOS/GOARCH (e.g. linux/amd64). May be repeated.
                                              If omitted, builds all supported combinations:
                                              darwin:  amd64, arm64
                                              linux:   amd64, arm64, 386
                                              windows: amd64, arm64, 386
USAGE
}

function validate::platforms() {
  local platform os arch

  for platform in "$@"; do
    if [[ "${platform}" != *"/"* ]] || [[ "${platform}" == *"/"*"/"* ]]; then
      util::print::error "invalid platform \"${platform}\" (expected OS/ARCH with a single slash, e.g. linux/amd64)"
    fi

    os="${platform%%/*}"
    arch="${platform##*/}"
    if [[ -z "${os}" || -z "${arch}" ]]; then
      util::print::error "invalid platform \"${platform}\" (OS and ARCH must be non-empty)"
    fi

    case "${os}" in
      darwin|linux|windows) ;;
      *)
        util::print::error "invalid OS \"${os}\" in \"${platform}\" (supported: darwin, linux, windows)"
        ;;
    esac

    case "${arch}" in
      amd64|arm64|386) ;;
      *)
        util::print::error "invalid ARCH \"${arch}\" in \"${platform}\" (supported: amd64, arm64, 386)"
        ;;
    esac
  done
}

function repo::prepare() {
  util::print::title "Preparing repo..."

  rm -rf "${BUILD_DIR}"
  mkdir -p "${BUILD_DIR}"
}

function build::archives() {
  local version
  version="${1}"
  shift

  local platforms=("$@")
  local platform os arch binary archive name
  local work_dir staging_dir

  work_dir="$(mktemp -d)"
  trap 'rm -rf "${work_dir:-}"' EXIT

  for platform in "${platforms[@]}"; do
    os="${platform%%/*}"
    arch="${platform##*/}"
    binary="libpak-tools"
    if [[ "${os}" == "windows" ]]; then
      binary="${binary}.exe"
    fi

    util::print::info "Building libpak-tools for ${os}/${arch}"

    (
      cd "${ROOT_DIR}"
      GOOS="${os}" \
      GOARCH="${arch}" \
      CGO_ENABLED=0 \
        go build \
          -ldflags "-s -w -X github.com/paketo-buildpacks/libpak-tools/commands.version=${version}" \
          -o "${work_dir}/${binary}" \
          main.go
    )

    staging_dir="${work_dir}/${os}-${arch}"
    mkdir -p "${staging_dir}"
    cp "${ROOT_DIR}/LICENSE" "${ROOT_DIR}/README.md" "${work_dir}/${binary}" "${staging_dir}/"

    name="$(archive::name "${os}" "${arch}")"
    archive="${BUILD_DIR}/${name}"

    util::print::info "Packaging ${name}"

    if [[ "${os}" == "windows" ]]; then
      (
        cd "${staging_dir}"
        zip -q "${archive}" LICENSE README.md "${binary}"
      )
    else
      (
        cd "${staging_dir}"
        tar -czf "${archive}" LICENSE README.md "${binary}"
      )
    fi
  done

  rm -rf "${work_dir}"
  trap - EXIT
}

function archive::name() {
  local os arch os_label arch_label extension

  os="${1}"
  arch="${2}"

  case "${os}" in
    darwin)
      os_label="Darwin"
      extension="tar.gz"
      ;;

    linux)
      os_label="Linux"
      extension="tar.gz"
      ;;

    windows)
      os_label="Windows"
      extension="zip"
      ;;
  esac

  case "${arch}" in
    amd64)
      arch_label="x86_64"
      ;;

    386)
      arch_label="i386"
      ;;

    *)
      arch_label="${arch}"
      ;;
  esac

  echo "libpak-tools_${os_label}_${arch_label}.${extension}"
}

function checksums::write() {
  local version archive
  version="${1}"
  local archives=()

  util::print::title "Writing checksums..."

  (
    cd "${BUILD_DIR}"

    for archive in libpak-tools_*.tar.gz libpak-tools_*.zip; do
      if [[ -f "${archive}" ]]; then
        archives+=("${archive}")
      fi
    done

    if command -v sha256sum > /dev/null 2>&1; then
      sha256sum "${archives[@]}" > "libpak-tools_${version}_checksums.txt"
    else
      shasum -a 256 "${archives[@]}" > "libpak-tools_${version}_checksums.txt"
    fi
  )
}

main "${@:-}"
