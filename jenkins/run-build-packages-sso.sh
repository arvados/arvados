#!/bin/bash

. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados SSO package

Syntax:
        WORKSPACE=/path/to/arvados-sso $(basename $0) [options]

Options:

--build-bundle-packages  (default: false)
    Build package with vendor/bundle included
--debug
    Output debug information (default: false)
--target
    Distribution to build packages for (default: debian7)

WORKSPACE=path         Path to the Arvados SSO source tree to build packages from

EOF

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}
BUILD_BUNDLE_PACKAGES=0
TARGET=debian7

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,build-bundle-packages,debug,target: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --target)
            TARGET="$2"; shift
            ;;
        --debug)
            DEBUG=1
            ;;
        --build-bundle-packages)
            BUILD_BUNDLE_PACKAGES=1
            ;;
        --)
            if [ $# -gt 1 ]; then
                echo >&2 "$0: unrecognized argument '$2'. Try: $0 --help"
                exit 1
            fi
            ;;
    esac
    shift
done

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
fi

case "$TARGET" in
    debian7)
        FORMAT=deb
        ;;
    debian8)
        FORMAT=deb
        ;;
    ubuntu1204)
        FORMAT=deb
        ;;
    ubuntu1404)
        FORMAT=deb
        ;;
    centos6)
        FORMAT=rpm
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac

if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

if ! [[ -d "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: $WORKSPACE is not a directory"
  echo >&2
  exit 1
fi

# Test for fpm
fpm --version >/dev/null 2>&1

if [[ "$?" != 0 ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: fpm not found"
  echo >&2
  exit 1
fi

RUN_BUILD_PACKAGES_PATH="`dirname \"$0\"`"
RUN_BUILD_PACKAGES_PATH="`( cd \"$RUN_BUILD_PACKAGES_PATH\" && pwd )`"  # absolutized and normalized
if [ -z "$RUN_BUILD_PACKAGES_PATH" ] ; then
  # error; for some reason, the path is not accessible
  # to the script (e.g. permissions re-evaled after suid)
  exit 1  # fail
fi

debug_echo "$0 is running from $RUN_BUILD_PACKAGES_PATH"
debug_echo "Workspace is $WORKSPACE"

if [[ -f /etc/profile.d/rvm.sh ]]; then
    source /etc/profile.d/rvm.sh
    GEM="rvm-exec default gem"
else
    GEM=gem
fi

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
# No executables in the sso server package
#find -type d -name 'bin' |xargs -I {} find {} -type f |xargs -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

debug_echo "umask is" `umask`

if [[ ! -d "$WORKSPACE/packages/$TARGET" ]]; then
  mkdir -p $WORKSPACE/packages/$TARGET
fi

# Build the SSO server package

cd "$WORKSPACE"

SSO_VERSION=$(version_from_git)
PACKAGE_NAME=arvados-sso

if [[ ! -d "$WORKSPACE/tmp" ]]; then
  mkdir $WORKSPACE/tmp
fi

if [[ "$BUILD_BUNDLE_PACKAGES" != 0 ]]; then
  bundle install --path vendor/bundle >"$STDOUT_IF_DEBUG"
fi

/usr/bin/git rev-parse HEAD > git-commit.version

cd $WORKSPACE/packages/$TARGET

# Annoyingly, we require a database.yml file for rake assets:precompile to work. So for now,
# we do that in the upgrade script.
# TODO: add bogus database.yml file so we can precompile the assets and put them in the
# package. Then remove that database.yml file again. It has to be a valid file though.
#RAILS_ENV=production RAILS_GROUPS=assets bundle exec rake assets:precompile

# There are just 2 excludes left here, all the others are pulled in via fpm-info.sh.
# The .git directory is excluded by git implicitly, so we can't pick it up from .gitignore.
# The packages directory needs to be explictly excluded here because it will only be listed
# if it exists at the time fpm-info.sh runs. If it does not exist at that time, this script
# will create it and when fpm runs, it will include the directory. So we add it to the exclude
# list explicitly here, just in case.
declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "--vendor='Curoverse, Inc.'" "--url='https://arvados.org'" "--description='Arvados SSO server - Arvados is a free and open source platform for big data science.'" "--license='Expat License'" "-s" "dir" "-t" "$FORMAT" "-v" "$SSO_VERSION" "-x" "var/www/arvados-sso/current/.git" "-x" "var/www/arvados-sso/current/packages" "--after-install=$RUN_BUILD_PACKAGES_PATH/arvados-sso-server-extras/postinst.sh")

if [[ "$BUILD_BUNDLE_PACKAGES" != 0 ]]; then
  # This is the complete package with vendor/bundle included.
  # It's big, so we do not build it by default.
  COMMAND_ARR+=("-n" "${PACKAGE_NAME}-with-bundle")
else
  # The default package excludes vendor/bundle
  COMMAND_ARR+=("-n" "${PACKAGE_NAME}" "-x" "var/www/arvados-sso/current/vendor/bundle")
fi

# Append --depends X and other arguments specified by fpm-info.sh in
# the package source dir. These are added last so they can override
# the arguments added by this script.
declare -a fpm_args=()
declare -a fpm_depends=()
FPM_INFO="$WORKSPACE/fpm-info.sh"
if [[ -e "$FPM_INFO" ]]; then
  debug_echo "Loading fpm overrides from $FPM_INFO"
  source "$FPM_INFO"
fi

for i in "${fpm_depends[@]}"; do
  COMMAND_ARR+=('--depends' "$i")
done
COMMAND_ARR+=("${fpm_args[@]}")
COMMAND_ARR+=("$WORKSPACE/=/var/www/arvados-sso/current" "$RUN_BUILD_PACKAGES_PATH/arvados-sso-server-extras/arvados-sso-server-upgrade.sh=/usr/local/bin/arvados-sso-server-upgrade.sh")
debug_echo -e "\n${COMMAND_ARR[@]}\n"

FPM_RESULTS=$("${COMMAND_ARR[@]}")
FPM_EXIT_CODE=$?
fpm_verify $FPM_EXIT_CODE $FPM_RESULTS

# SSO server package build done

exit $EXITCODE
