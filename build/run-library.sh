#!/bin/bash

# A library of functions shared by the various scripts in this directory.

# This is the timestamp about when we merged changed to include licenses
# with Arvados packages.  We use it as a heuristic to add revisions for
# older packages.
LICENSE_PACKAGE_TS=20151208015500

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
    local format="$1"; shift
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
    local gem_name="$1"; shift
    local gem_version="$(nohash_version_from_git)"
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
    local license_file="${1:-agpl-3.0.txt}"; shift

    debug_echo "package_go_binary $src_path as $prog"

    local basename="${src_path##*/}"

    mkdir -p "$GOPATH/src/git.curoverse.com"
    ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

    cd "$GOPATH/src/git.curoverse.com/arvados.git/$src_path"
    local version="$(version_from_git)"
    local timestamp="$(timestamp_from_git)"

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

    declare -a switches=()
    systemd_unit="$WORKSPACE/${src_path}/${prog}.service"
    if [[ -e "${systemd_unit}" ]]; then
        switches+=(
            --after-install "${WORKSPACE}/build/go-package-scripts/postinst"
            --before-remove "${WORKSPACE}/build/go-package-scripts/prerm"
            "${systemd_unit}=/lib/systemd/system/${prog}.service")
    fi
    switches+=("$WORKSPACE/${license_file}=/usr/share/doc/$prog/${license_file}")

    fpm_build "$GOPATH/bin/${basename}=/usr/bin/${prog}" "${prog}" 'Curoverse, Inc.' dir "${version}" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=${description}" "${switches[@]}"
}

default_iteration() {
    local package_name="$1"; shift
    local package_version="$1"; shift
    local iteration=1
    if [[ $package_version =~ ^0\.1\.([0-9]{14})(\.|$) ]] && \
           [[ ${BASH_REMATCH[1]} -le $LICENSE_PACKAGE_TS ]]; then
        iteration=2
    fi
    echo $iteration
}

_build_rails_package_scripts() {
    local pkgname="$1"; shift
    local destdir="$1"; shift
    local srcdir="$RUN_BUILD_PACKAGES_PATH/rails-package-scripts"
    for scriptname in postinst prerm postrm; do
        cat "$srcdir/$pkgname.sh" "$srcdir/step2.sh" "$srcdir/$scriptname.sh" \
            >"$destdir/$scriptname" || return $?
    done
}

handle_rails_package() {
    local pkgname="$1"; shift
    local srcdir="$1"; shift
    local license_path="$1"; shift
    local scripts_dir="$(mktemp --tmpdir -d "$pkgname-XXXXXXXX.scripts")" && \
    local version_file="$(mktemp --tmpdir "$pkgname-XXXXXXXX.version")" && (
        set -e
        _build_rails_package_scripts "$pkgname" "$scripts_dir"
        cd "$srcdir"
        mkdir -p tmp
        version_from_git >"$version_file"
        git rev-parse HEAD >git-commit.version
        bundle package --all
    )
    if [[ 0 != "$?" ]] || ! cd "$WORKSPACE/packages/$TARGET"; then
        echo "ERROR: $pkgname package prep failed" >&2
        rm -rf "$scripts_dir" "$version_file"
        EXITCODE=1
        return 1
    fi
    local railsdir="/var/www/${pkgname%-server}/current"
    local -a pos_args=("$srcdir/=$railsdir" "$pkgname" "Curoverse, Inc." dir
                       "$(cat "$version_file")")
    local license_arg="$license_path=$railsdir/$(basename "$license_path")"
    local -a switches=(--iteration=6
                       --after-install "$scripts_dir/postinst"
                       --before-remove "$scripts_dir/prerm"
                       --after-remove "$scripts_dir/postrm")
    # For some reason fpm excludes need to not start with /.
    local exclude_root="${railsdir#/}"
    # .git and packages are for the SSO server, which is built from its
    # repository root.
    local -a exclude_list=(.git packages tmp log coverage Capfile\* \
                           config/deploy\* config/application.yml)
    # for arvados-workbench, we need to have the (dummy) config/database.yml in the package
    if  [[ "$pkgname" != "arvados-workbench" ]]; then
      exclude_list+=('config/database.yml')
    fi
    for exclude in ${exclude_list[@]}; do
        switches+=(-x "$exclude_root/$exclude")
    done
    fpm_build "${pos_args[@]}" "${switches[@]}" \
              -x "$exclude_root/vendor/bundle" "$@" "$license_arg"
    rm -rf "$scripts_dir" "$version_file"
}

# Build packages for everything
fpm_build () {
  # The package source.  Depending on the source type, this can be a
  # path, or the name of the package in an upstream repository (e.g.,
  # pip).
  PACKAGE=$1
  shift
  # The name of the package to build.
  PACKAGE_NAME=$1
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

  local default_iteration_value="$(default_iteration "$PACKAGE" "$VERSION")"

  case "$PACKAGE_TYPE" in
      python)
          # All Arvados Python2 packages depend on Python 2.7.
          # Make sure we build with that for consistency.
          set -- "$@" --python-bin python2.7 \
              --python-easyinstall "$EASY_INSTALL2" \
              --python-package-name-prefix "$PYTHON2_PKG_PREFIX" \
              --prefix "$PYTHON2_PREFIX" \
              --python-install-lib "$PYTHON2_INSTALL_LIB" \
              --exclude "${PYTHON2_INSTALL_LIB#/}/tests" \
              --depends "$PYTHON2_PACKAGE"
          # Fix --iteration for #9242.
          default_iteration_value=$(($default_iteration_value + 1))
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
              --prefix "$PYTHON3_PREFIX" \
              --python-install-lib "$PYTHON3_INSTALL_LIB" \
              --exclude "${PYTHON3_INSTALL_LIB#/}/tests" \
              --depends "$PYTHON3_PACKAGE"
          # Fix --iteration for #9242.
          default_iteration_value=$(($default_iteration_value + 1))
          ;;
  esac

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "-s" "$PACKAGE_TYPE" "-t" "$FORMAT")
  if [ python = "$PACKAGE_TYPE" ] && [ deb = "$FORMAT" ]; then
      # Dependencies are built from setup.py.  Since setup.py will never
      # refer to Debian package iterations, it doesn't make sense to
      # enforce those in the .deb dependencies.
      COMMAND_ARR+=(--deb-ignore-iteration-in-dependencies)
  fi

  if [[ "${DEBUG:-0}" != "0" ]]; then
    COMMAND_ARR+=('--verbose' '--log' 'info')
  fi

  if [[ -n "$PACKAGE_NAME" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  if [[ "$VERSION" != "" ]]; then
    COMMAND_ARR+=('-v' "$VERSION")
  fi
  # We can always add an --iteration here.  If another one is specified in $@,
  # that will take precedence, as desired.
  COMMAND_ARR+=(--iteration "$default_iteration_value")

  # Append --depends X and other arguments specified by fpm-info.sh in
  # the package source dir. These are added last so they can override
  # the arguments added by this script.
  declare -a fpm_args=()
  declare -a build_depends=()
  declare -a fpm_depends=()
  declare -a fpm_exclude=()
  declare -a fpm_dirs=(
      # source dir part of 'dir' package ("/source=/dest" => "/source"):
      "${PACKAGE%%=/*}"
      # backports ("llfuse==0.41.1" => "backports/python-llfuse")
      "${WORKSPACE}/backports/${PACKAGE_TYPE}-${PACKAGE%%[<=>]*}")
  if [[ -n "$PACKAGE_NAME" ]]; then
      fpm_dirs+=("${WORKSPACE}/backports/${PACKAGE_NAME}")
  fi
  for pkgdir in "${fpm_dirs[@]}"; do
      fpminfo="$pkgdir/fpm-info.sh"
      if [[ -e "$fpminfo" ]]; then
          debug_echo "Loading fpm overrides from $fpminfo"
          source "$fpminfo"
          break
      fi
  done
  for pkg in "${build_depends[@]}"; do
      if [[ $TARGET =~ debian|ubuntu ]]; then
          pkg_deb=$(ls "$WORKSPACE/packages/$TARGET/$pkg_"*.deb | sort -rg | awk 'NR==1')
          if [[ -e $pkg_deb ]]; then
              echo "Installing build_dep $pkg from $pkg_deb"
              dpkg -i "$pkg_deb"
          else
              echo "Attemping to install build_dep $pkg using apt-get"
              apt-get install -y "$pkg"
          fi
          apt-get -y -f install
      else
          pkg_rpm=$(ls "$WORKSPACE/packages/$TARGET/$pkg"-[0-9]*.rpm | sort -rg | awk 'NR==1')
          if [[ -e $pkg_rpm ]]; then
              echo "Installing build_dep $pkg from $pkg_rpm"
              rpm -i "$pkg_rpm"
          else
              echo "Attemping to install build_dep $pkg"
              rpm -i "$pkg"
          fi
      fi
  done
  for i in "${fpm_depends[@]}"; do
    COMMAND_ARR+=('--depends' "$i")
  done
  for i in "${fpm_exclude[@]}"; do
    COMMAND_ARR+=('--exclude' "$i")
  done

  # Append remaining function arguments directly to fpm's command line.
  for i; do
    COMMAND_ARR+=("$i")
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

title () {
    txt="********** $1 **********"
    printf "\n%*s%s\n\n" $((($COLUMNS-${#txt})/2)) "" "$txt"
}

checkexit() {
    if [[ "$1" != "0" ]]; then
        title "!!!!!! $2 FAILED !!!!!!"
        failures+=("$2 (`timer`)")
    else
        successes+=("$2 (`timer`)")
    fi
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    echo -n "$(($SECONDS - $t0))s"
}

report_outcomes() {
    for x in "${successes[@]}"
    do
        echo "Pass: $x"
    done

    if [[ ${#failures[@]} == 0 ]]
    then
        echo "All test suites passed."
    else
        echo "Failures (${#failures[@]}):"
        for x in "${failures[@]}"
        do
            echo "Fail: $x"
        done
    fi
}
