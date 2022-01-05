#!/bin/bash -xe
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# A library of functions shared by the various scripts in this directory.

# This is the timestamp about when we merged changed to include licenses
# with Arvados packages.  We use it as a heuristic to add revisions for
# older packages.
LICENSE_PACKAGE_TS=20151208015500

if [[ -z "$ARVADOS_BUILDING_VERSION" ]]; then
    RAILS_PACKAGE_ITERATION=1
else
    RAILS_PACKAGE_ITERATION="$ARVADOS_BUILDING_ITERATION"
fi

debug_echo () {
    echo "$@" >"$STDOUT_IF_DEBUG"
}

find_python_program() {
    prog="$1"
    shift
    for prog in "$@"; do
        if "$prog" --version >/dev/null 2>&1; then
            echo "$prog"
            return 0
        fi
    done
    cat >&2 <<EOF
$helpmessage

Error: $prog (from Python setuptools module) not found

EOF
    exit 1
}

format_last_commit_here() {
    local format="$1"; shift
    local dir="${1:-.}"; shift
    TZ=UTC git log -n1 --first-parent "--format=format:$format" "$dir"
}

version_from_git() {
    # Output the version being built, or if we're building a
    # dev/prerelease, output a version number based on the git log for
    # the given $subdir.
    local subdir="$1"; shift
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_VERSION"
        return
    fi

    local git_ts git_hash
    declare $(format_last_commit_here "git_ts=%ct git_hash=%h" "$subdir")
    ARVADOS_BUILDING_VERSION="$($WORKSPACE/build/version-at-commit.sh $git_hash)"
    echo "$ARVADOS_BUILDING_VERSION"
}

nohash_version_from_git() {
    local subdir="$1"; shift
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_VERSION"
        return
    fi
    version_from_git $subdir | cut -d. -f1-4
}

timestamp_from_git() {
    local subdir="$1"; shift
    format_last_commit_here "%ct" "$subdir"
}

calculate_python_sdk_cwl_package_versions() {
  python_sdk_version=$(cd sdk/python && python3 arvados_version.py)
  cwl_runner_version=$(cd sdk/cwl && python3 arvados_version.py)
}

# Usage: get_native_arch
get_native_arch() {
  # Only amd64 and aarch64 are supported at the moment
  local native_arch=""
  case "$HOSTTYPE" in
    x86_64)
      native_arch="amd64"
      ;;
    aarch64)
      native_arch="arm64"
      ;;
    *)
      echo "Error: architecture not supported"
      exit 1
      ;;
  esac
  echo $native_arch
}

handle_ruby_gem() {
    local gem_name="$1"; shift
    local gem_version="$(nohash_version_from_git)"
    local gem_src_dir="$(pwd)"

    if [[ -n "$ONLY_BUILD" ]] && [[ "$gem_name" != "$ONLY_BUILD" ]] ; then
        return 0
    fi

    if ! [[ -e "${gem_name}-${gem_version}.gem" ]]; then
        find -maxdepth 1 -name "${gem_name}-*.gem" -delete

        # -q appears to be broken in gem version 2.2.2
        $GEM build "$gem_name.gemspec" $DASHQ_UNLESS_DEBUG >"$STDOUT_IF_DEBUG" 2>"$STDERR_IF_DEBUG"
    fi
}

calculate_go_package_version() {
  # $__returnvar has the nameref attribute set, which means it is a reference
  # to another variable that is passed in as the first argument to this function.
  # see https://www.gnu.org/software/bash/manual/html_node/Shell-Parameters.html
  local -n __returnvar="$1"; shift
  local oldpwd="$PWD"

  cd "$WORKSPACE"
  go mod download

  # Update the version number and build a new package if the vendor
  # bundle has changed, or the command imports anything from the
  # Arvados SDK and the SDK has changed.
  declare -a checkdirs=(go.mod go.sum)
  while [ -n "$1" ]; do
      checkdirs+=("$1")
      shift
  done
  # Even our rails packages (version calculation happens here!) depend on a go component (arvados-server)
  # Everything depends on the build directory.
  checkdirs+=(sdk/go lib build)
  local timestamp=0
  for dir in ${checkdirs[@]}; do
      cd "$WORKSPACE"
      ts="$(timestamp_from_git "$dir")"
      if [[ "$ts" -gt "$timestamp" ]]; then
          version=$(version_from_git "$dir")
          timestamp="$ts"
      fi
  done
  cd "$oldpwd"
  __returnvar="$version"
}

# Usage: package_go_binary services/foo arvados-foo [deb|rpm] [amd64|arm64] "Compute foo to arbitrary precision" [apache-2.0.txt]
package_go_binary() {
  local src_path="$1"; shift
  local prog="$1"; shift
  local package_format="$1"; shift
  local target_arch="$1"; shift
  local description="$1"; shift
  local license_file="${1:-agpl-3.0.txt}"; shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "$prog" != "$ONLY_BUILD" ]]; then
    # arvados-workbench depends on arvados-server at build time, so even when
    # only arvados-workbench is being built, we need to build arvados-server too
    if [[ "$prog" != "arvados-server" ]] || [[ "$ONLY_BUILD" != "arvados-workbench" ]]; then
      debug_echo -e "Skipping build of $prog package."
      return 0
    fi
  fi

  native_arch=$(get_native_arch)

  if [[ "$native_arch" != "amd64" ]] && [[ -n "$target_arch" ]] && [[ "$native_arch" != "$target_arch" ]]; then
    echo "Error: no cross compilation support for Go on $native_arch yet, can not build $prog for $target_arch"
    return 1
  fi

  if [[ -n "$target_arch" ]]; then
    # A target architecture has been specified
    package_go_binary_worker "$src_path" "$prog" "$package_format" "$description" "$native_arch" "$target_arch" "$license_file"
    return $?
  else
    # No target architecture specified, default to native target. When on amd64 also crosscompile arm64
    # but only when building deb packages (centos does not have support for crosscompiling userspace).
    archs=($native_arch)
    if [[ "$native_arch" == "amd64" ]] && [[ "$package_format" == "deb" ]]; then
      archs+=("arm64")
    fi
    for ta in ${archs[@]}; do
      package_go_binary_worker "$src_path" "$prog" "$package_format" "$description" "$native_arch" "$ta" "$license_file"
      retval=$?
      if [[ $retval -ne 0 ]]; then
        return $retval
      fi
    done
  fi
}

# Usage: package_go_binary services/foo arvados-foo deb "Compute foo to arbitrary precision" [amd64/arm64] [amd64/arm64] [apache-2.0.txt]
package_go_binary_worker() {
    local src_path="$1"; shift
    local prog="$1"; shift
    local package_format="$1"; shift
    local description="$1"; shift
    local native_arch="${1:-amd64}"; shift
    local target_arch="${1:-amd64}"; shift
    local license_file="${1:-agpl-3.0.txt}"; shift

    if [[ "$native_arch" != "$target_arch" ]] && [[ "$package_format" == "rpm" ]]; then
      echo "Error: no cross compilation support for Go on $native_arch ($package_format), can not build $prog for $target_arch"
      return 1
    fi

    debug_echo "package_go_binary $src_path as $prog (native arch: $native_arch, target arch: $target_arch)"
    local basename="${src_path##*/}"
    calculate_go_package_version go_package_version $src_path

    cd $WORKSPACE/packages/$TARGET
    test_package_presence "$prog" "$go_package_version" "go" "" "$target_arch"
    if [[ $? -ne 0 ]]; then
      return 0
    fi

    echo "Building $package_format ($target_arch) package for $prog from $src_path"
    if [[ "$native_arch" == "amd64" ]] && [[ "$target_arch" == "arm64" ]]; then
      CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc GOARCH=${target_arch} go get -ldflags "-X git.arvados.org/arvados.git/lib/cmd.version=${go_package_version} -X main.version=${go_package_version}" "git.arvados.org/arvados.git/$src_path"
    else
      GOARCH=${arch} go get -ldflags "-X git.arvados.org/arvados.git/lib/cmd.version=${go_package_version} -X main.version=${go_package_version}" "git.arvados.org/arvados.git/$src_path"
    fi

    local -a switches=()

    binpath=$GOPATH/bin/${basename}
    if [[ "${target_arch}" != "${native_arch}" ]]; then
      switches+=("-a${target_arch}")
      binpath="$GOPATH/bin/linux_${target_arch}/${basename}"
    fi

    systemd_unit="$WORKSPACE/${src_path}/${prog}.service"
    if [[ -e "${systemd_unit}" ]]; then
        switches+=(
            --after-install "${WORKSPACE}/build/go-python-package-scripts/postinst"
            --before-remove "${WORKSPACE}/build/go-python-package-scripts/prerm"
            "${systemd_unit}=/lib/systemd/system/${prog}.service")
    fi
    switches+=("$WORKSPACE/${license_file}=/usr/share/doc/$prog/${license_file}")

    fpm_build "${WORKSPACE}/${src_path}" "$binpath=/usr/bin/${prog}" "${prog}" dir "${go_package_version}" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=${description}" "${switches[@]}"
}

# Usage: package_go_so lib/foo arvados_foo.so arvados-foo deb amd64 "Arvados foo library"
package_go_so() {
    local src_path="$1"; shift
    local sofile="$1"; shift
    local pkg="$1"; shift
    local package_format="$1"; shift
    local target_arch="$1"; shift # supported: amd64, arm64
    local description="$1"; shift

    if [[ -n "$ONLY_BUILD" ]] && [[ "$pkg" != "$ONLY_BUILD" ]]; then
      debug_echo -e "Skipping build of $pkg package."
      return 0
    fi

    debug_echo "package_go_so $src_path as $pkg"

    calculate_go_package_version go_package_version $src_path
    cd $WORKSPACE/packages/$TARGET
    test_package_presence $pkg $go_package_version go || return 1
    cd $WORKSPACE/$src_path
    go build -buildmode=c-shared -o ${GOPATH}/bin/${sofile}
    cd $WORKSPACE/packages/$TARGET
    local -a fpmargs=(
        "--url=https://arvados.org"
        "--license=Apache License, Version 2.0"
        "--description=${description}"
        "$WORKSPACE/apache-2.0.txt=/usr/share/doc/$pkg/apache-2.0.txt"
    )
    if [[ -e "$WORKSPACE/$src_path/pam-configs-arvados" ]]; then
        fpmargs+=("$WORKSPACE/$src_path/pam-configs-arvados=/usr/share/doc/$pkg/pam-configs-arvados-go")
    fi
    if [[ -e "$WORKSPACE/$src_path/README" ]]; then
        fpmargs+=("$WORKSPACE/$src_path/README=/usr/share/doc/$pkg/README")
    fi
    fpm_build "${WORKSPACE}/${src_path}" "$GOPATH/bin/${sofile}=/usr/lib/${sofile}" "${pkg}" dir "${go_package_version}" "${fpmargs[@]}"
}

default_iteration() {
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_ITERATION"
        return
    fi
    local package_name="$1"; shift
    local package_version="$1"; shift
    local package_type="$1"; shift
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

rails_package_version() {
    local pkgname="$1"; shift
    local srcdir="$1"; shift
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_VERSION"
        return
    fi
    local version="$(version_from_git)"
    if [ $pkgname = "arvados-api-server" -o $pkgname = "arvados-workbench" ] ; then
        calculate_go_package_version version cmd/arvados-server "$srcdir"
    fi
    echo $version
}

test_rails_package_presence() {
  local pkgname="$1"; shift
  local srcdir="$1"; shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "$pkgname" != "$ONLY_BUILD" ]] ; then
    return 1
  fi

  tmppwd=`pwd`

  cd $srcdir

  local version="$(rails_package_version "$pkgname" "$srcdir")"

  cd $tmppwd

  test_package_presence $pkgname $version rails "$RAILS_PACKAGE_ITERATION"
}

get_complete_package_name() {
  # if the errexit flag is set, unset it until this function returns
  # otherwise, the shift calls below will abort the program if optional arguments are not supplied
  if [ -o errexit ]; then
    set +e
    trap 'set -e' RETURN
  fi
  # $__returnvar has the nameref attribute set, which means it is a reference
  # to another variable that is passed in as the first argument to this function.
  # see https://www.gnu.org/software/bash/manual/html_node/Shell-Parameters.html
  local -n __returnvar="$1"; shift
  local pkgname="$1"; shift
  local version="$1"; shift
  local pkgtype="$1"; shift
  local iteration="$1"; shift
  local arch="$1"; shift
  if [[ "$iteration" == "" ]]; then
      iteration="$(default_iteration "$pkgname" "$version" "$pkgtype")"
  fi

  if [[ "$arch" == "" ]]; then
    native_arch=$(get_native_arch)
    rpm_native_arch="x86_64"
    if [[ "$HOSTTYPE" == "aarch64" ]]; then
      rpm_native_arch="arm64"
    fi
    rpm_architecture="$rpm_native_arch"
    deb_architecture="$native_arch"

    if [[ "$pkgtype" =~ ^(src)$ ]]; then
      rpm_architecture="noarch"
      deb_architecture="all"
    fi
  else
    rpm_architecture=$arch
    deb_architecture=$arch
  fi

  local complete_pkgname="${pkgname}_$version${iteration:+-$iteration}_$deb_architecture.deb"
  if [[ "$FORMAT" == "rpm" ]]; then
      # rpm packages get iteration 1 if we don't supply one
      iteration=${iteration:-1}
      complete_pkgname="$pkgname-$version-${iteration}.$rpm_architecture.rpm"
  fi
  __returnvar=${complete_pkgname}
}

# Test if the package already exists, if not return 0, if it does return 1
test_package_presence() {
    local pkgname="$1"; shift
    local version="$1"; shift
    local pkgtype="$1"; shift
    local iteration="$1"; shift
    local arch="$1"; shift
    if [[ -n "$ONLY_BUILD" ]] && [[ "$pkgname" != "$ONLY_BUILD" ]] ; then
      # arvados-workbench depends on arvados-server at build time, so even when
      # only arvados-workbench is being built, we need to build arvados-server too
      if [[ "$pkgname" != "arvados-server" ]] || [[ "$ONLY_BUILD" != "arvados-workbench" ]]; then
        return 1
      fi
    fi

    local full_pkgname
    get_complete_package_name full_pkgname "$pkgname" "$version" "$pkgtype" "$iteration" "$arch"

    # See if we can skip building the package, only if it already exists in the
    # processed/ directory. If so, move it back to the packages directory to make
    # sure it gets picked up by the test and/or upload steps.
    # Get the list of packages from the repos

    if [[ "$FORCE_BUILD" == "1" ]]; then
      echo "Package $full_pkgname build forced with --force-build, building"
    elif [[ "$FORMAT" == "deb" ]]; then
      declare -A dd
      dd[debian10]=buster
      dd[debian11]=bullseye
      dd[ubuntu1804]=bionic
      dd[ubuntu2004]=focal
      D=${dd[$TARGET]}
      if [ ${pkgname:0:3} = "lib" ]; then
        repo_subdir=${pkgname:0:4}
      else
        repo_subdir=${pkgname:0:1}
      fi

      repo_pkg_list=$(curl -s -o - http://apt.arvados.org/${D}/pool/main/${repo_subdir}/${pkgname}/)
      echo "${repo_pkg_list}" |grep -q ${full_pkgname}
      if [ $? -eq 0 ] ; then
        echo "Package $full_pkgname exists upstream, not rebuilding, downloading instead!"
        curl -s -o "$WORKSPACE/packages/$TARGET/${full_pkgname}" http://apt.arvados.org/${D}/pool/main/${repo_subdir}/${pkgname}/${full_pkgname}
        return 1
      elif test -f "$WORKSPACE/packages/$TARGET/processed/${full_pkgname}" ; then
        echo "Package $full_pkgname exists, not rebuilding!"
        return 1
      else
        echo "Package $full_pkgname not found, building"
        return 0
      fi
    else
      centos_repo="http://rpm.arvados.org/CentOS/7/dev/x86_64/"

      repo_pkg_list=$(curl -s -o - ${centos_repo})
      echo ${repo_pkg_list} |grep -q ${full_pkgname}
      if [ $? -eq 0 ]; then
        echo "Package $full_pkgname exists upstream, not rebuilding, downloading instead!"
        curl -s -o "$WORKSPACE/packages/$TARGET/${full_pkgname}" ${centos_repo}${full_pkgname}
        return 1
      elif test -f "$WORKSPACE/packages/$TARGET/processed/${full_pkgname}" ; then
        echo "Package $full_pkgname exists, not rebuilding!"
        return 1
      else
        echo "Package $full_pkgname not found, building"
        return 0
      fi
    fi
}

handle_rails_package() {
    local pkgname="$1"; shift

    if [[ -n "$ONLY_BUILD" ]] && [[ "$pkgname" != "$ONLY_BUILD" ]] ; then
        return 0
    fi
    local srcdir="$1"; shift
    cd "$srcdir"
    local license_path="$1"; shift
    local version="$(rails_package_version "$pkgname" "$srcdir")"
    echo "$version" >package-build.version
    local scripts_dir="$(mktemp --tmpdir -d "$pkgname-XXXXXXXX.scripts")" && \
    (
        set -e
        _build_rails_package_scripts "$pkgname" "$scripts_dir"
        cd "$srcdir"
        mkdir -p tmp
        git rev-parse HEAD >git-commit.version
        bundle package --all
    )
    if [[ 0 != "$?" ]] || ! cd "$WORKSPACE/packages/$TARGET"; then
        echo "ERROR: $pkgname package prep failed" >&2
        rm -rf "$scripts_dir"
        EXITCODE=1
        return 1
    fi
    local railsdir="/var/www/${pkgname%-server}/current"
    local -a pos_args=("$srcdir/=$railsdir" "$pkgname" dir "$version")
    local license_arg="$license_path=$railsdir/$(basename "$license_path")"
    local -a switches=(--after-install "$scripts_dir/postinst"
                       --before-remove "$scripts_dir/prerm"
                       --after-remove "$scripts_dir/postrm")
    if [[ -z "$ARVADOS_BUILDING_VERSION" ]]; then
        switches+=(--iteration $RAILS_PACKAGE_ITERATION)
    fi
    # For some reason fpm excludes need to not start with /.
    local exclude_root="${railsdir#/}"
    local -a exclude_list=(tmp log coverage Capfile\* \
                           config/deploy\* config/application.yml)
    # for arvados-workbench, we need to have the (dummy) config/database.yml in the package
    if  [[ "$pkgname" != "arvados-workbench" ]]; then
      exclude_list+=('config/database.yml')
    fi
    for exclude in ${exclude_list[@]}; do
        switches+=(-x "$exclude_root/$exclude")
    done
    fpm_build "${srcdir}" "${pos_args[@]}" "${switches[@]}" \
              -x "$exclude_root/vendor/cache-*" \
              -x "$exclude_root/vendor/bundle" "$@" "$license_arg"
    rm -rf "$scripts_dir"
}

# Usage: handle_api_server [amd64|arm64]
handle_api_server () {
  local target_arch="${1:-amd64}"; shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "$ONLY_BUILD" != "arvados-api-server" ]] ; then
    debug_echo -e "Skipping build of arvados-api-server package."
    return 0
  fi

  native_arch=$(get_native_arch)
  if [[ "$target_arch" != "$native_arch" ]]; then
    echo "Error: no cross compilation support for Rails yet, can not build arvados-api-server for $ARCH"
    echo
    exit 1
  fi

  # Build the API server package
  test_rails_package_presence arvados-api-server "$WORKSPACE/services/api"
  if [[ "$?" == "0" ]]; then
    calculate_go_package_version arvados_server_version cmd/arvados-server
    arvados_server_iteration=$(default_iteration "arvados-server" "$arvados_server_version" "go")
    handle_rails_package arvados-api-server "$WORKSPACE/services/api" \
        "$WORKSPACE/agpl-3.0.txt" --url="https://arvados.org" \
        --description="Arvados API server - Arvados is a free and open source platform for big data science." \
        --license="GNU Affero General Public License, version 3.0" --depends "arvados-server = ${arvados_server_version}-${arvados_server_iteration}"
  fi
}

# Usage: handle_workbench [amd64|arm64]
handle_workbench () {
  local target_arch="${1:-amd64}"; shift
  if [[ -n "$ONLY_BUILD" ]] && [[ "$ONLY_BUILD" != "arvados-workbench" ]] ; then
    debug_echo -e "Skipping build of arvados-workbench package."
    return 0
  fi

  native_arch=$(get_native_arch)
  if [[ "$target_arch" != "$native_arch" ]]; then
    echo "Error: no cross compilation support for Rails yet, can not build arvados-workbench for $native_arch"
    echo
    exit 1
  fi

  if [[ "$native_arch" != "amd64" ]]; then
    echo "Error: building the arvados-workbench package is not yet supported on this architecture ($native_arch)."
    echo
    exit 1
  fi

  # Build the workbench server package
  test_rails_package_presence arvados-workbench "$WORKSPACE/apps/workbench"
  if [[ "$?" == "0" ]] ; then
    (
        set -e

        calculate_go_package_version arvados_server_version cmd/arvados-server
        arvados_server_iteration=$(default_iteration "arvados-server" "$arvados_server_version" "go")

        # The workbench package has a build-time dependency on the arvados-server
        # package for config manipulation, so install it first.
        cd $WORKSPACE/cmd/arvados-server
        get_complete_package_name arvados_server_pkgname arvados-server ${arvados_server_version} go

        arvados_server_pkg_path="$WORKSPACE/packages/$TARGET/${arvados_server_pkgname}"
        if [[ ! -e ${arvados_server_pkg_path} ]]; then
          arvados_server_pkg_path="$WORKSPACE/packages/$TARGET/processed/${arvados_server_pkgname}"
        fi
        if [[ "$FORMAT" == "deb" ]]; then
          dpkg -i ${arvados_server_pkg_path}
        else
          rpm -i ${arvados_server_pkg_path}
        fi

        cd "$WORKSPACE/apps/workbench"

        # We need to bundle to be ready even when we build a package without vendor directory
        # because asset compilation requires it.
        bundle install --system >"$STDOUT_IF_DEBUG"

        # clear the tmp directory; the asset generation step will recreate tmp/cache/assets,
        # and we want that in the package, so it's easier to not exclude the tmp directory
        # from the package - empty it instead.
        rm -rf tmp
        mkdir tmp

        # Set up an appropriate config.yml
        arvados-server config-dump -config <(cat /etc/arvados/config.yml 2>/dev/null || echo  "Clusters: {zzzzz: {}}") > /tmp/x
        mkdir -p /etc/arvados/
        mv /tmp/x /etc/arvados/config.yml
        perl -p -i -e 'BEGIN{undef $/;} s/WebDAV(.*?):\n( *)ExternalURL: ""/WebDAV$1:\n$2ExternalURL: "example.com"/g' /etc/arvados/config.yml

        ARVADOS_CONFIG=none RAILS_ENV=production RAILS_GROUPS=assets bin/rake npm:install >"$STDOUT_IF_DEBUG"
        ARVADOS_CONFIG=none RAILS_ENV=production RAILS_GROUPS=assets bin/rake assets:precompile >"$STDOUT_IF_DEBUG"

        # Remove generated configuration files so they don't go in the package.
        rm -rf /etc/arvados/
    )

    if [[ "$?" != "0" ]]; then
      echo "ERROR: Asset precompilation failed"
      EXITCODE=1
    else
      handle_rails_package arvados-workbench "$WORKSPACE/apps/workbench" \
          "$WORKSPACE/agpl-3.0.txt" --url="https://arvados.org" \
          --description="Arvados Workbench - Arvados is a free and open source platform for big data science." \
          --license="GNU Affero General Public License, version 3.0" --depends "arvados-server = ${arvados_server_version}-${arvados_server_iteration}"
    fi
  fi
}

# Usage: handle_cwltest [deb|rpm] [amd64|arm64]
handle_cwltest () {
  local package_format="$1"; shift
  local target_arch="${1:-amd64}"; shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "$ONLY_BUILD" != "python3-cwltest" ]] ; then
    debug_echo -e "Skipping build of cwltest package."
    return 0
  fi
  cd "$WORKSPACE"
  if [[ -e "$WORKSPACE/cwltest" ]]; then
    rm -rf "$WORKSPACE/cwltest"
  fi
  git clone https://github.com/common-workflow-language/cwltest.git
  # signal to our build script that we want a cwltest executable installed in /usr/bin/
  mkdir cwltest/bin && touch cwltest/bin/cwltest
  fpm_build_virtualenv "cwltest" "cwltest" "$package_format" "$target_arch"
  # The python->python3 metapackage
  build_metapackage "cwltest" "cwltest"
  cd "$WORKSPACE"
  rm -rf "$WORKSPACE/cwltest"
}

# Usage: handle_arvados_src
handle_arvados_src () {
  if [[ -n "$ONLY_BUILD" ]] && [[ "$ONLY_BUILD" != "arvados-src" ]] ; then
    debug_echo -e "Skipping build of arvados-src package."
    return 0
  fi
  # arvados-src
  (
      cd "$WORKSPACE"
      COMMIT_HASH=$(format_last_commit_here "%H")
      arvados_src_version="$(version_from_git)"

      cd $WORKSPACE/packages/$TARGET
      test_package_presence arvados-src "$arvados_src_version" src ""

      if [[ "$?" == "0" ]]; then
        cd "$WORKSPACE"
        SRC_BUILD_DIR=$(mktemp -d)
        # mktemp creates the directory with 0700 permissions by default
        chmod 755 $SRC_BUILD_DIR
        git clone $DASHQ_UNLESS_DEBUG "$WORKSPACE/.git" "$SRC_BUILD_DIR"
        cd "$SRC_BUILD_DIR"

        # go into detached-head state
        git checkout $DASHQ_UNLESS_DEBUG "$COMMIT_HASH"
        echo "$COMMIT_HASH" >git-commit.version

        cd $WORKSPACE/packages/$TARGET
        fpm_build "$WORKSPACE" $SRC_BUILD_DIR/=/usr/local/arvados/src arvados-src 'dir' "$arvados_src_version" "--exclude=usr/local/arvados/src/.git" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

        rm -rf "$SRC_BUILD_DIR"
      fi
  )
}

# Usage: handle_libarvados_perl
handle_libarvados_perl () {
  if [[ -n "$ONLY_BUILD" ]] || [[ "$ONLY_BUILD" != "libarvados-perl" ]] ; then
    debug_echo -e "Skipping build of libarvados-perl package."
    return 0
  fi
  cd "$WORKSPACE/sdk/perl"
  libarvados_perl_version="$(version_from_git)"

  cd $WORKSPACE/packages/$TARGET
  test_package_presence libarvados-perl "$libarvados_perl_version"

  if [[ "$?" == "0" ]]; then
    cd "$WORKSPACE/sdk/perl"

    if [[ -e Makefile ]]; then
      make realclean >"$STDOUT_IF_DEBUG"
    fi
    find -maxdepth 1 \( -name 'MANIFEST*' -or -name "libarvados-perl*.$FORMAT" \) \
        -delete
    rm -rf install

    perl Makefile.PL INSTALL_BASE=install >"$STDOUT_IF_DEBUG" && \
        make install INSTALLDIRS=perl >"$STDOUT_IF_DEBUG" && \
        fpm_build "$WORKSPACE/sdk/perl" install/lib/=/usr/share libarvados-perl \
        dir "$(version_from_git)" install/man/=/usr/share/man \
        "$WORKSPACE/apache-2.0.txt=/usr/share/doc/libarvados-perl/apache-2.0.txt" && \
        mv --no-clobber libarvados-perl*.$FORMAT "$WORKSPACE/packages/$TARGET/"
  fi
}

# Build python packages with a virtualenv built-in
# Usage: fpm_build_virtualenv arvados-python-client sdk/python [deb|rpm] [amd64|arm64]
fpm_build_virtualenv () {
  local pkg=$1; shift
  local pkg_dir=$1; shift
  local package_format="$1"; shift
  local target_arch="${1:-amd64}"; shift

  native_arch=$(get_native_arch)

  if [[ "$pkg" != "arvados-docker-cleaner" ]]; then
    PYTHON_PKG=$PYTHON3_PKG_PREFIX-$pkg
  else
    # Exception to our package naming convention
    PYTHON_PKG=$pkg
  fi

  # arvados-python-client sdist should always be built, to be available
  # for other dependent packages.
  if [[ -n "$ONLY_BUILD" ]] && [[ "arvados-python-client" != "$pkg" ]] && [[ "$PYTHON_PKG" != "$ONLY_BUILD" ]] && [[ "$pkg" != "$ONLY_BUILD" ]]; then
    debug_echo -e "Skipping build of $pkg package."
    return 0
  fi

  if [[ -n "$target_arch" ]] && [[ "$native_arch" == "$target_arch" ]]; then
      fpm_build_virtualenv_worker "$pkg" "$pkg_dir" "$package_format" "$native_arch" "$target_arch"
  elif [[ -z "$target_arch" ]]; then
    fpm_build_virtualenv_worker "$pkg" "$pkg_dir" "$package_format" "$native_arch" "$native_arch"
  else
    echo "Error: no cross compilation support for Python yet, can not build $pkg for $target_arch"
    return 1
  fi
}

# Build python packages with a virtualenv built-in
# Usage: fpm_build_virtualenv_worker arvados-python-client sdk/python python3 [deb|rpm] [amd64|arm64] [amd64|arm64]
fpm_build_virtualenv_worker () {
  PKG=$1; shift
  PKG_DIR=$1; shift
  local package_format="$1"; shift
  local native_arch="${1:-amd64}"; shift
  local target_arch=${1:-amd64}; shift

  # Set up
  STDOUT_IF_DEBUG=/dev/null
  STDERR_IF_DEBUG=/dev/null
  DASHQ_UNLESS_DEBUG=-q
  if [[ "$DEBUG" != "0" ]]; then
      STDOUT_IF_DEBUG=/dev/stdout
      STDERR_IF_DEBUG=/dev/stderr
      DASHQ_UNLESS_DEBUG=
  fi
  if [[ "$ARVADOS_BUILDING_ITERATION" == "" ]]; then
    ARVADOS_BUILDING_ITERATION=1
  fi

  local python=python3
  pip=pip3
  PACKAGE_PREFIX=$PYTHON3_PKG_PREFIX

  if [[ "$PKG" != "arvados-docker-cleaner" ]]; then
    PYTHON_PKG=$PACKAGE_PREFIX-$PKG
  else
    # Exception to our package naming convention
    PYTHON_PKG=$PKG
  fi

  cd $WORKSPACE/$PKG_DIR

  rm -rf dist/*

  # Get the latest setuptools
  if ! $pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'; then
    echo "Error, unable to upgrade setuptools with"
    echo "  $pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'"
    exit 1
  fi
  # filter a useless warning (when building the cwltest package) from the stderr output
  if ! $python setup.py $DASHQ_UNLESS_DEBUG sdist 2> >(grep -v 'warning: no previously-included files matching'); then
    echo "Error, unable to run $python setup.py sdist for $PKG"
    exit 1
  fi

  PACKAGE_PATH=`(cd dist; ls *tar.gz)`

  if [[ "arvados-python-client" == "$PKG" ]]; then
    PYSDK_PATH=`pwd`/dist/
  fi

  if [[ -n "$ONLY_BUILD" ]] && [[ "$PYTHON_PKG" != "$ONLY_BUILD" ]] && [[ "$PKG" != "$ONLY_BUILD" ]]; then
    return 0
  fi

  # Determine the package version from the generated sdist archive
  if [[ -n "$ARVADOS_BUILDING_VERSION" ]] ; then
      UNFILTERED_PYTHON_VERSION=$ARVADOS_BUILDING_VERSION
      PYTHON_VERSION=$(echo -n $ARVADOS_BUILDING_VERSION | sed s/~dev/.dev/g | sed s/~rc/rc/g)
  else
      PYTHON_VERSION=$(awk '($1 == "Version:"){print $2}' *.egg-info/PKG-INFO)
      UNFILTERED_PYTHON_VERSION=$(echo -n $PYTHON_VERSION | sed s/\.dev/~dev/g |sed 's/\([0-9]\)rc/\1~rc/g')
  fi

  # See if we actually need to build this package; does it exist already?
  # We can't do this earlier than here, because we need PYTHON_VERSION...
  # This isn't so bad; the sdist call above is pretty quick compared to
  # the invocation of virtualenv and fpm, below.
  if ! test_package_presence "$PYTHON_PKG" "$UNFILTERED_PYTHON_VERSION" "$python" "$ARVADOS_BUILDING_ITERATION" "$target_arch"; then
    return 0
  fi

  echo "Building $package_format ($target_arch) package for $PKG from $PKG_DIR"

  # Package the sdist in a virtualenv
  echo "Creating virtualenv..."

  cd dist

  rm -rf build
  rm -f $PYTHON_PKG*deb
  echo "virtualenv version: `virtualenv --version`"
  virtualenv_command="virtualenv --python `which $python` $DASHQ_UNLESS_DEBUG build/usr/share/$python/dist/$PYTHON_PKG"

  if ! $virtualenv_command; then
    echo "Error, unable to run"
    echo "  $virtualenv_command"
    exit 1
  fi

  if ! build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U pip; then
    echo "Error, unable to upgrade pip with"
    echo "  build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U pip"
    exit 1
  fi
  echo "pip version:        `build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip --version`"

  if ! build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'; then
    echo "Error, unable to upgrade setuptools with"
    echo "  build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'"
    exit 1
  fi
  echo "setuptools version: `build/usr/share/$python/dist/$PYTHON_PKG/bin/$python -c 'import setuptools; print(setuptools.__version__)'`"

  if ! build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U wheel; then
    echo "Error, unable to upgrade wheel with"
    echo "  build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U wheel"
    exit 1
  fi
  echo "wheel version:      `build/usr/share/$python/dist/$PYTHON_PKG/bin/wheel version`"

  if [[ "$TARGET" != "centos7" ]] || [[ "$PYTHON_PKG" != "python-arvados-fuse" ]]; then
    build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -f $PYSDK_PATH $PACKAGE_PATH
  else
    # centos7 needs these special tweaks to install python-arvados-fuse
    build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG docutils
    PYCURL_SSL_LIBRARY=nss build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -f $PYSDK_PATH $PACKAGE_PATH
  fi

  if [[ "$?" != "0" ]]; then
    echo "Error, unable to run"
    echo "  build/usr/share/$python/dist/$PYTHON_PKG/bin/$pip install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -f $PYSDK_PATH $PACKAGE_PATH"
    exit 1
  fi

  cd build/usr/share/$python/dist/$PYTHON_PKG/

  # Replace the shebang lines in all python scripts, and handle the activate
  # scripts too. This is a functional replacement of the 237 line
  # virtualenv_tools.py script that doesn't work in python3 without serious
  # patching, minus the parts we don't need (modifying pyc files, etc).
  for binfile in `ls bin/`; do
    if ! file --mime bin/$binfile |grep -q binary; then
      # Not a binary file
      if [[ "$binfile" =~ ^activate(.csh|.fish|)$ ]]; then
        # these 'activate' scripts need special treatment
        sed -i "s/VIRTUAL_ENV=\".*\"/VIRTUAL_ENV=\"\/usr\/share\/$python\/dist\/$PYTHON_PKG\"/" bin/$binfile
        sed -i "s/VIRTUAL_ENV \".*\"/VIRTUAL_ENV \"\/usr\/share\/$python\/dist\/$PYTHON_PKG\"/" bin/$binfile
      else
        if grep -q -E '^#!.*/bin/python\d?' bin/$binfile; then
          # Replace shebang line
          sed -i "1 s/^.*$/#!\/usr\/share\/$python\/dist\/$PYTHON_PKG\/bin\/python/" bin/$binfile
        fi
      fi
    fi
  done

  cd - >$STDOUT_IF_DEBUG

  find build -iname '*.pyc' -exec rm {} \;
  find build -iname '*.pyo' -exec rm {} \;

  # Finally, generate the package
  echo "Creating package..."

  declare -a COMMAND_ARR=("fpm" "-s" "dir" "-t" "$package_format")

  if [[ -n "$target_arch" ]] && [[ "$target_arch" != "amd64" ]]; then
    COMMAND_ARR+=("-a$target_arch")
  fi

  if [[ "$MAINTAINER" != "" ]]; then
    COMMAND_ARR+=('--maintainer' "$MAINTAINER")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  COMMAND_ARR+=('--url' 'https://arvados.org')

  # Get description
  DESCRIPTION=`grep '\sdescription' $WORKSPACE/$PKG_DIR/setup.py|cut -f2 -d=|sed -e "s/[',\\"]//g"`
  COMMAND_ARR+=('--description' "$DESCRIPTION")

  # Get license string
  LICENSE_STRING=`grep license $WORKSPACE/$PKG_DIR/setup.py|cut -f2 -d=|sed -e "s/[',\\"]//g"`
  COMMAND_ARR+=('--license' "$LICENSE_STRING")

  if [[ "$package_format" == "rpm" ]]; then
    # Make sure to conflict with the old rh-python36 packages we used to publish
    COMMAND_ARR+=('--conflicts' "rh-python36-python-$PKG")
  fi

  if [[ "$DEBUG" != "0" ]]; then
    COMMAND_ARR+=('--verbose' '--log' 'info')
  fi

  COMMAND_ARR+=('-v' $(echo -n "$PYTHON_VERSION" | sed s/.dev/~dev/g | sed s/rc/~rc/g))
  COMMAND_ARR+=('--iteration' "$ARVADOS_BUILDING_ITERATION")
  COMMAND_ARR+=('-n' "$PYTHON_PKG")
  COMMAND_ARR+=('-C' "build")

  systemd_unit="$WORKSPACE/$PKG_DIR/$PKG.service"
  if [[ -e "${systemd_unit}" ]]; then
    COMMAND_ARR+=('--after-install' "${WORKSPACE}/build/go-python-package-scripts/postinst")
    COMMAND_ARR+=('--before-remove' "${WORKSPACE}/build/go-python-package-scripts/prerm")
  fi

  COMMAND_ARR+=('--depends' "$PYTHON3_PACKAGE")

  # avoid warning
  COMMAND_ARR+=('--deb-no-default-config-files')

  # Append --depends X and other arguments specified by fpm-info.sh in
  # the package source dir. These are added last so they can override
  # the arguments added by this script.
  declare -a fpm_args=()
  declare -a fpm_depends=()

  fpminfo="$WORKSPACE/$PKG_DIR/fpm-info.sh"
  if [[ -e "$fpminfo" ]]; then
    echo "Loading fpm overrides from $fpminfo"
    if ! source "$fpminfo"; then
      echo "Error, unable to source $WORKSPACE/$PKG_DIR/fpm-info.sh for $PKG"
      exit 1
    fi
  fi

  for i in "${fpm_depends[@]}"; do
    COMMAND_ARR+=('--depends' "$i")
  done

  for i in "${fpm_depends[@]}"; do
    COMMAND_ARR+=('--replaces' "python-$PKG")
  done

  # make sure the systemd service file ends up in the right place
  # used by arvados-docker-cleaner
  if [[ -e "${systemd_unit}" ]]; then
    COMMAND_ARR+=("usr/share/$python/dist/$PKG/share/doc/$PKG/$PKG.service=/lib/systemd/system/$PKG.service")
  fi

  COMMAND_ARR+=("${fpm_args[@]}")

  # Make sure to install all our package binaries in /usr/bin.
  # We have to walk $WORKSPACE/$PKG_DIR/bin rather than
  # $WORKSPACE/build/usr/share/$python/dist/$PYTHON_PKG/bin/ to get the list
  # because the latter also includes all the python binaries for the virtualenv.
  # We have to take the copies of our binaries from the latter directory, though,
  # because those are the ones we rewrote the shebang line of, above.
  if [[ -e "$WORKSPACE/$PKG_DIR/bin" ]]; then
    for binary in `ls $WORKSPACE/$PKG_DIR/bin`; do
      COMMAND_ARR+=("usr/share/$python/dist/$PYTHON_PKG/bin/$binary=/usr/bin/")
    done
  fi

  # the python3-arvados-cwl-runner package comes with cwltool, expose that version
  if [[ -e "$WORKSPACE/$PKG_DIR/dist/build/usr/share/$python/dist/$PYTHON_PKG/bin/cwltool" ]]; then
    COMMAND_ARR+=("usr/share/$python/dist/$PYTHON_PKG/bin/cwltool=/usr/bin/")
  fi

  COMMAND_ARR+=(".")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  # if something went wrong and debug is off, print out the fpm command that errored
  if ! fpm_verify $FPM_EXIT_CODE $FPM_RESULTS && [[ "$STDOUT_IF_DEBUG" == "/dev/null" ]]; then
    echo "fpm returned an error executing the command:"
    echo
    echo -e "\n${COMMAND_ARR[@]}\n"
  else
    echo `ls *$package_format`
    mv $WORKSPACE/$PKG_DIR/dist/*$package_format $WORKSPACE/packages/$TARGET/
  fi
  echo
}

# build_metapackage builds meta packages that help with the python to python 3 package migration
build_metapackage() {
  # base package name (e.g. arvados-python-client)
  BASE_NAME=$1
  shift
  PKG_DIR=$1
  shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "python-$BASE_NAME" != "$ONLY_BUILD" ]]; then
    return 0
  fi

  if [[ "$ARVADOS_BUILDING_ITERATION" == "" ]]; then
    ARVADOS_BUILDING_ITERATION=1
  fi

  if [[ -z "$ARVADOS_BUILDING_VERSION" ]]; then
    cd $WORKSPACE/$PKG_DIR
    pwd
    rm -rf dist/*

    # Get the latest setuptools
    if ! pip3 install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'; then
      echo "Error, unable to upgrade setuptools with XY"
      echo "  pip3 install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'"
      exit 1
    fi
    # filter a useless warning (when building the cwltest package) from the stderr output
    if ! python3 setup.py $DASHQ_UNLESS_DEBUG sdist 2> >(grep -v 'warning: no previously-included files matching'); then
      echo "Error, unable to run python3 setup.py sdist for $PKG"
      exit 1
    fi

    PYTHON_VERSION=$(awk '($1 == "Version:"){print $2}' *.egg-info/PKG-INFO)
    UNFILTERED_PYTHON_VERSION=$(echo -n $PYTHON_VERSION | sed s/\.dev/~dev/g |sed 's/\([0-9]\)rc/\1~rc/g')

  else
    UNFILTERED_PYTHON_VERSION=$ARVADOS_BUILDING_VERSION
    PYTHON_VERSION=$(echo -n $ARVADOS_BUILDING_VERSION | sed s/~dev/.dev/g | sed s/~rc/rc/g)
  fi

  cd - >$STDOUT_IF_DEBUG
  if [[ -d "$BASE_NAME" ]]; then
    rm -rf $BASE_NAME
  fi
  mkdir $BASE_NAME
  cd $BASE_NAME

  if [[ "$FORMAT" == "deb" ]]; then
    cat >ns-control <<EOF
Section: misc
Priority: optional
Standards-Version: 3.9.2

Package: python-${BASE_NAME}
Version: ${PYTHON_VERSION}-${ARVADOS_BUILDING_ITERATION}
Maintainer: Arvados Package Maintainers <packaging@arvados.org>
Depends: python3-${BASE_NAME}
Description: metapackage to ease the upgrade to the Pyhon 3 version of ${BASE_NAME}
 This package is a metapackage that will automatically install the new version of
 ${BASE_NAME} which is Python 3 based and has a different name.
EOF

    /usr/bin/equivs-build ns-control
    if [[ $? -ne 0 ]]; then
      echo "Error running 'equivs-build ns-control', is the 'equivs' package installed?"
      return 1
    fi
  elif [[ "$FORMAT" == "rpm" ]]; then
    cat >meta.spec <<EOF
Summary: metapackage to ease the upgrade to the Python 3 version of ${BASE_NAME}
Name: python-${BASE_NAME}
Version: ${PYTHON_VERSION}
Release: ${ARVADOS_BUILDING_ITERATION}
License: distributable

Requires: python3-${BASE_NAME}

%description
This package is a metapackage that will automatically install the new version of
python-${BASE_NAME} which is Python 3 based and has a different name.

%prep

%build

%clean

%install

%post

%files


%changelog
* Mon Apr 12 2021 Arvados Package Maintainers <packaging@arvados.org>
- initial release
EOF

    /usr/bin/rpmbuild -ba meta.spec
    if [[ $? -ne 0 ]]; then
      echo "Error running 'rpmbuild -ba meta.spec', is the 'rpm-build' package installed?"
      return 1
    else
      mv /root/rpmbuild/RPMS/x86_64/python-${BASE_NAME}*.${FORMAT} .
      if [[ $? -ne 0 ]]; then
        echo "Error finding rpm file output of 'rpmbuild -ba meta.spec'"
        return 1
      fi
    fi
  else
    echo "Unknown format"
    return 1
  fi

  if [[ $EXITCODE -ne 0 ]]; then
    return 1
  else
    echo `ls *$FORMAT`
    mv *$FORMAT $WORKSPACE/packages/$TARGET/
  fi

  # clean up
  cd - >$STDOUT_IF_DEBUG
  if [[ -d "$BASE_NAME" ]]; then
    rm -rf $BASE_NAME
  fi
}

# Build packages for everything
fpm_build () {
  # Source dir where fpm-info.sh (if any) will be found.
  SRC_DIR=$1
  shift
  # The package source.  Depending on the source type, this can be a
  # path, or the name of the package in an upstream repository (e.g.,
  # pip).
  PACKAGE=$1
  shift
  # The name of the package to build.
  PACKAGE_NAME=$1
  shift
  # The type of source package.  Passed to fpm -s.  Default "dir".
  PACKAGE_TYPE=${1:-dir}
  shift
  # Optional: the package version number.  Passed to fpm -v.
  VERSION=$1
  shift

  if [[ -n "$ONLY_BUILD" ]] && [[ "$PACKAGE_NAME" != "$ONLY_BUILD" ]] && [[ "$PACKAGE" != "$ONLY_BUILD" ]] ; then
    # arvados-workbench depends on arvados-server at build time, so even when
    # only arvados-workbench is being built, we need to build arvados-server too
    if [[ "$PACKAGE_NAME" != "arvados-server" ]] || [[ "$ONLY_BUILD" != "arvados-workbench" ]]; then
      return 0
    fi
  fi

  local default_iteration_value="$(default_iteration "$PACKAGE" "$VERSION" "$PACKAGE_TYPE")"

  declare -a COMMAND_ARR=("fpm" "-s" "$PACKAGE_TYPE" "-t" "$FORMAT")
  if [ python = "$PACKAGE_TYPE" ] && [ deb = "$FORMAT" ]; then
      # Dependencies are built from setup.py.  Since setup.py will never
      # refer to Debian package iterations, it doesn't make sense to
      # enforce those in the .deb dependencies.
      COMMAND_ARR+=(--deb-ignore-iteration-in-dependencies)
  fi

  if [[ "$DEBUG" != "0" ]]; then
    COMMAND_ARR+=('--verbose' '--log' 'info')
  fi

  if [[ -n "$PACKAGE_NAME" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$MAINTAINER" != "" ]]; then
    COMMAND_ARR+=('--maintainer' "$MAINTAINER")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  if [[ "$VERSION" != "" ]]; then
    COMMAND_ARR+=('-v' "$VERSION")
  fi
  if [[ -n "$default_iteration_value" ]]; then
      # We can always add an --iteration here.  If another one is specified in $@,
      # that will take precedence, as desired.
      COMMAND_ARR+=(--iteration "$default_iteration_value")
  fi

  # Append --depends X and other arguments specified by fpm-info.sh in
  # the package source dir. These are added last so they can override
  # the arguments added by this script.
  declare -a fpm_args=()
  declare -a build_depends=()
  declare -a fpm_depends=()
  declare -a fpm_conflicts=()
  declare -a fpm_exclude=()
  if [[ ! -d "$SRC_DIR" ]]; then
      echo >&2 "BUG: looking in wrong dir for fpm-info.sh: $pkgdir"
      exit 1
  fi
  fpminfo="${SRC_DIR}/fpm-info.sh"
  if [[ -e "$fpminfo" ]]; then
      debug_echo "Loading fpm overrides from $fpminfo"
      source "$fpminfo"
  fi
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
  for i in "${fpm_conflicts[@]}"; do
    COMMAND_ARR+=('--conflicts' "$i")
  done
  for i in "${fpm_exclude[@]}"; do
    COMMAND_ARR+=('--exclude' "$i")
  done

  COMMAND_ARR+=("${fpm_args[@]}")

  # Append remaining function arguments directly to fpm's command line.
  for i; do
    COMMAND_ARR+=("$i")
  done

  COMMAND_ARR+=("$PACKAGE")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  fpm_verify $FPM_EXIT_CODE $FPM_RESULTS

  # if something went wrong and debug is off, print out the fpm command that errored
  if [[ 0 -ne $? ]] && [[ "$STDOUT_IF_DEBUG" == "/dev/null" ]]; then
    echo -e "\n${COMMAND_ARR[@]}\n"
  fi
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
    echo
    echo "Error: $PACKAGE: Unable to figure out package name from fpm results:"
    echo
    echo $FPM_RESULTS
    echo
    return 1
  elif [[ "$FPM_RESULTS" =~ "File already exists" ]]; then
    echo "Package $FPM_PACKAGE_NAME exists, not rebuilding"
    return 0
  elif [[ 0 -ne "$FPM_EXIT_CODE" ]]; then
    EXITCODE=1
    echo "Error building package for $1:\n $FPM_RESULTS"
    return 1
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

title() {
    printf '%s %s\n' "=======" "$1"
}

checkexit() {
    if [[ "$1" != "0" ]]; then
        title "$2 -- FAILED"
        failures+=("$2 (`timer`)")
    else
        successes+=("$2 (`timer`)")
    fi
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    if [[ -n "$t0" ]]; then
        echo -n "$(($SECONDS - $t0))s"
    fi
}

report_outcomes() {
    for x in "${successes[@]}"
    do
        echo "Pass: $x"
    done

    if [[ ${#failures[@]} == 0 ]]
    then
        if [[ ${#successes[@]} != 0 ]]; then
           echo "All test suites passed."
        fi
    else
        echo "Failures (${#failures[@]}):"
        for x in "${failures[@]}"
        do
            echo "Fail: $x"
        done
    fi
}
