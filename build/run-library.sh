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

# Usage: package_go_binary services/foo arvados-foo "Compute foo to arbitrary precision" [apache-2.0.txt]
package_go_binary() {
    local src_path="$1"; shift
    local prog="$1"; shift
    local description="$1"; shift
    local license_file="${1:-agpl-3.0.txt}"; shift

    if [[ -n "$ONLY_BUILD" ]] && [[ "$prog" != "$ONLY_BUILD" ]]; then
      # arvados-workbench depends on arvados-server at build time, so even when
      # only arvados-workbench is being built, we need to build arvados-server too
      if [[ "$prog" != "arvados-server" ]] || [[ "$ONLY_BUILD" != "arvados-workbench" ]]; then
        return 0
      fi
    fi

    debug_echo "package_go_binary $src_path as $prog"

    local basename="${src_path##*/}"
    calculate_go_package_version go_package_version $src_path

    cd $WORKSPACE/packages/$TARGET
    test_package_presence $prog $go_package_version go

    if [[ "$?" != "0" ]]; then
      return 1
    fi

    go get -ldflags "-X git.arvados.org/arvados.git/lib/cmd.version=${go_package_version} -X main.version=${go_package_version}" "git.arvados.org/arvados.git/$src_path"

    local -a switches=()
    systemd_unit="$WORKSPACE/${src_path}/${prog}.service"
    if [[ -e "${systemd_unit}" ]]; then
        switches+=(
            --after-install "${WORKSPACE}/build/go-python-package-scripts/postinst"
            --before-remove "${WORKSPACE}/build/go-python-package-scripts/prerm"
            "${systemd_unit}=/lib/systemd/system/${prog}.service")
    fi
    switches+=("$WORKSPACE/${license_file}=/usr/share/doc/$prog/${license_file}")

    fpm_build "${WORKSPACE}/${src_path}" "$GOPATH/bin/${basename}=/usr/bin/${prog}" "${prog}" dir "${go_package_version}" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=${description}" "${switches[@]}"
}

# Usage: package_go_so lib/foo arvados_foo.so arvados-foo "Arvados foo library"
package_go_so() {
    local src_path="$1"; shift
    local sofile="$1"; shift
    local pkg="$1"; shift
    local description="$1"; shift

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
    rpm_architecture="x86_64"
    deb_architecture="amd64"

    if [[ "$pkgtype" =~ ^(src)$ ]]; then
      rpm_architecture="noarch"
      deb_architecture="all"
    fi

    # These python packages have binary components
    if [[ "$pkgname" =~ (ruamel|ciso|pycrypto|pyyaml) ]]; then
      rpm_architecture="x86_64"
      deb_architecture="amd64"
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
    get_complete_package_name full_pkgname $pkgname $version $pkgtype $iteration $arch

    # See if we can skip building the package, only if it already exists in the
    # processed/ directory. If so, move it back to the packages directory to make
    # sure it gets picked up by the test and/or upload steps.
    # Get the list of packages from the repos

    if [[ "$FORCE_BUILD" == "1" ]]; then
      echo "Package $full_pkgname build forced with --force-build, building"
    elif [[ "$FORMAT" == "deb" ]]; then
      declare -A dd
      dd[debian10]=buster
      dd[ubuntu1604]=xenial
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

# Build python packages with a virtualenv built-in
fpm_build_virtualenv () {
  PKG=$1
  shift
  PKG_DIR=$1
  shift
  PACKAGE_TYPE=${1:-python}
  shift

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

  local python=""
  case "$PACKAGE_TYPE" in
    python3)
        python=python3
        pip=pip3
        PACKAGE_PREFIX=$PYTHON3_PKG_PREFIX
        ;;
  esac

  if [[ "$PKG" != "arvados-docker-cleaner" ]]; then
    PYTHON_PKG=$PACKAGE_PREFIX-$PKG
  else
    # Exception to our package naming convention
    PYTHON_PKG=$PKG
  fi

  # arvados-python-client sdist should always be built, to be available
  # for other dependent packages.
  if [[ -n "$ONLY_BUILD" ]] && [[ "arvados-python-client" != "$PKG" ]] && [[ "$PYTHON_PKG" != "$ONLY_BUILD" ]] && [[ "$PKG" != "$ONLY_BUILD" ]]; then
    return 0
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
  if ! test_package_presence "$PYTHON_PKG" $UNFILTERED_PYTHON_VERSION $PACKAGE_TYPE $ARVADOS_BUILDING_ITERATION; then
    return 0
  fi

  echo "Building $FORMAT package for $PKG from $PKG_DIR"

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

  declare -a COMMAND_ARR=("fpm" "-s" "dir" "-t" "$FORMAT")

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
  if [[ -e "$WORKSPACE/$PKG_DIR/dist/build/usr/share/$python/dist/python-arvados-cwl-runner/bin/cwltool" ]]; then
    COMMAND_ARR+=("usr/share/$python/dist/python-arvados-cwl-runner/bin/cwltool=/usr/bin/")
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
    echo `ls *$FORMAT`
    mv $WORKSPACE/$PKG_DIR/dist/*$FORMAT $WORKSPACE/packages/$TARGET/
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
