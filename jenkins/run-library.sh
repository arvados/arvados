#!/bin/bash

# A library of functions shared by the various scripts in this directory.

debug_echo () {
    echo "$@" >"$STDOUT_IF_DEBUG"
}

find_easy_install() {
    for version_suffix in "$@"; do
        if "easy_install$version_suffix" --version >/dev/null 2>&1; then
            echo "easy_install$version_suffix"
            return 0
        fi
    done
    cat >&2 <<EOF
$helpmessage

Error: easy_install$1 (from Python setuptools module) not found

EOF
    exit 1
}

format_last_commit_here() {
    local format=$1; shift
    TZ=UTC git log -n1 --first-parent "--format=format:$format" .
}

version_from_git() {
  # Generates a version number from the git log for the current working
  # directory, and writes it to stdout.
  local git_ts git_hash
  declare $(format_last_commit_here "git_ts=%ct git_hash=%h")
  echo "0.1.$(date -ud "@$git_ts" +%Y%m%d%H%M%S).$git_hash"
}

nohash_version_from_git() {
    version_from_git | cut -d. -f1-3
}

timestamp_from_git() {
    format_last_commit_here "%ct"
}

handle_python_package () {
  # This function assumes the current working directory is the python package directory
  if [ -n "$(find dist -name "*-$(nohash_version_from_git).tar.gz" -print -quit)" ]; then
    # This package doesn't need rebuilding.
    return
  fi
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  python setup.py $DASHQ_UNLESS_DEBUG sdist
}

handle_ruby_gem() {
    local gem_name=$1; shift
    local gem_version=$(nohash_version_from_git)
    local gem_src_dir="$(pwd)"

    if ! [[ -e "${gem_name}-${gem_version}.gem" ]]; then
        find -maxdepth 1 -name "${gem_name}-*.gem" -delete

        # -q appears to be broken in gem version 2.2.2
        $GEM build "$gem_name.gemspec" $DASHQ_UNLESS_DEBUG >"$STDOUT_IF_DEBUG" 2>"$STDERR_IF_DEBUG"
    fi
}

# Usage: package_go_binary services/foo arvados-foo "Compute foo to arbitrary precision"
package_go_binary() {
    local src_path="$1"; shift
    local prog="$1"; shift
    local description="$1"; shift

    debug_echo "package_go_binary $src_path as $prog"

    local basename="${src_path##*/}"

    mkdir -p "$GOPATH/src/git.curoverse.com"
    ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

    cd "$GOPATH/src/git.curoverse.com/arvados.git/$src_path"
    local version=$(version_from_git)
    local timestamp=$(timestamp_from_git)

    # If the command imports anything from the Arvados SDK, bump the
    # version number and build a new package whenever the SDK changes.
    if grep -qr git.curoverse.com/arvados .; then
        cd "$GOPATH/src/git.curoverse.com/arvados.git/sdk/go"
        if [[ $(timestamp_from_git) -gt "$timestamp" ]]; then
            version=$(version_from_git)
        fi
    fi

    cd $WORKSPACE/packages/$TARGET
    go get "git.curoverse.com/arvados.git/$src_path"
    fpm_build "$GOPATH/bin/$basename=/usr/bin/$prog" "$prog" 'Curoverse, Inc.' dir "$version" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=$description"
}

# Build packages for everything
fpm_build () {
  # The package source.  Depending on the source type, this can be a
  # path, or the name of the package in an upstream repository (e.g.,
  # pip).
  PACKAGE=$1
  shift
  # The name of the package to build.  Defaults to $PACKAGE.
  PACKAGE_NAME=${1:-$PACKAGE}
  shift
  # Optional: the vendor of the package.  Should be "Curoverse, Inc." for
  # packages of our own software.  Passed to fpm --vendor.
  VENDOR=$1
  shift
  # The type of source package.  Passed to fpm -s.  Default "python".
  PACKAGE_TYPE=${1:-python}
  shift
  # Optional: the package version number.  Passed to fpm -v.
  VERSION=$1
  shift

  case "$PACKAGE_TYPE" in
      python)
          # All Arvados Python2 packages depend on Python 2.7.
          # Make sure we build with that for consistency.
          set -- "$@" --python-bin python2.7 \
              --python-easyinstall "$EASY_INSTALL2" \
              --python-package-name-prefix "$PYTHON2_PKG_PREFIX" \
              --depends "$PYTHON2_PACKAGE"
          ;;
      python3)
          # fpm does not actually support a python3 package type.  Instead
          # we recognize it as a convenience shortcut to add several
          # necessary arguments to fpm's command line later, after we're
          # done handling positional arguments.
          PACKAGE_TYPE=python
          set -- "$@" --python-bin python3 \
              --python-easyinstall "$EASY_INSTALL3" \
              --python-package-name-prefix "$PYTHON3_PKG_PREFIX" \
              --depends "$PYTHON3_PACKAGE"
          ;;
  esac

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "-s" "$PACKAGE_TYPE" "-t" "$FORMAT")
  if [ python = "$PACKAGE_TYPE" ]; then
    COMMAND_ARR+=(--exclude=\*/{dist,site}-packages/tests/\*)
  fi

  if [[ "$PACKAGE_NAME" != "$PACKAGE" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  if [[ "$VERSION" != "" ]]; then
    COMMAND_ARR+=('-v' "$VERSION")
  fi

  # Append remaining function arguments directly to fpm's command line.
  for i; do
    COMMAND_ARR+=("$i")
  done

  # Append --depends X and other arguments specified by fpm-info.sh in
  # the package source dir. These are added last so they can override
  # the arguments added by this script.
  declare -a fpm_args=()
  declare -a fpm_depends=()
  if [[ -d "$PACKAGE" ]]; then
      FPM_INFO="$PACKAGE/fpm-info.sh"
  else
      FPM_INFO="${WORKSPACE}/backports/${PACKAGE_TYPE}-${PACKAGE}/fpm-info.sh"
  fi
  if [[ -e "$FPM_INFO" ]]; then
      debug_echo "Loading fpm overrides from $FPM_INFO"
      source "$FPM_INFO"
  fi
  for i in "${fpm_depends[@]}"; do
    COMMAND_ARR+=('--depends' "$i")
  done
  COMMAND_ARR+=("${fpm_args[@]}")

  COMMAND_ARR+=("$PACKAGE")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  fpm_verify $FPM_EXIT_CODE $FPM_RESULTS
}

# verify build results
fpm_verify () {
  FPM_EXIT_CODE=$1
  shift
  FPM_RESULTS=$@

  FPM_PACKAGE_NAME=''
  if [[ $FPM_RESULTS =~ ([A-Za-z0-9_\.-]*\.)(deb|rpm) ]]; then
    FPM_PACKAGE_NAME=${BASH_REMATCH[1]}${BASH_REMATCH[2]}
  fi

  if [[ "$FPM_PACKAGE_NAME" == "" ]]; then
    EXITCODE=1
    echo "Error: $PACKAGE: Unable to figure out package name from fpm results:"
    echo
    echo $FPM_RESULTS
    echo
  elif [[ "$FPM_RESULTS" =~ "File already exists" ]]; then
    echo "Package $FPM_PACKAGE_NAME exists, not rebuilding"
  elif [[ 0 -ne "$FPM_EXIT_CODE" ]]; then
    echo "Error building package for $1:\n $FPM_RESULTS"
  fi
}

install_package() {
  PACKAGES=$@
  if [[ "$FORMAT" == "deb" ]]; then
    $SUDO apt-get install $PACKAGES --yes
  elif [[ "$FORMAT" == "rpm" ]]; then
    $SUDO yum -q -y install $PACKAGES
  fi
}
