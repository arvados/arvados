#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This code runs after package variable definitions.

set -e

for DISTRO_FAMILY in $(. /etc/os-release && echo "${ID:-} ${ID_LIKE:-}"); do
    case "$DISTRO_FAMILY" in
        debian)
            RESETUP_CMD="dpkg-reconfigure $PACKAGE_NAME"
            break ;;
        rhel)
            RESETUP_CMD="dnf reinstall $PACKAGE_NAME"
            break ;;
    esac
done
if [ -z "$RESETUP_CMD" ]; then
   echo "$PACKAGE_NAME postinst skipped: don't recognize the distribution from /etc/os-release" >&2
   exit 0
fi
# This will be set to a command path after we install the version we need.
BUNDLE=

# systemd_ctl is just "systemctl if we booted with systemd, otherwise a noop."
# This makes the package installable in Docker containers, albeit without any
# service deployment.
if [ -d /run/systemd/system ]; then
    systemd_ctl() { systemctl "$@"; }
else
    systemd_ctl() { true; }
fi

systemd_quote() {
    if [ $# -ne 1 ]; then
        echo "error: systemd_quote requires exactly one argument" >&2
        return 2
    fi
    # See systemd.syntax(7) - Use double quotes with backslash escapes
    echo "$1" | sed -re 's/[\\"]/\\\0/g; s/^/"/; s/$/"/'
}

run_and_report() {
    # Usage: run_and_report ACTION_MSG CMD
    # This is the usual wrapper that prints ACTION_MSG, runs CMD, then writes
    # a message about whether CMD succeeded or failed.  Returns the exit code
    # of CMD.
    local action_message="$1"; shift
    local retcode=0
    echo -n "$action_message..."
    if "$@"; then
        echo " done."
    else
        retcode=$?
        echo " failed."
    fi
    return $retcode
}

report_not_ready() {
    local exitcode="$1"; shift
    local reason="$1"; shift
    local doc_url="${1:-}"
    case "$doc_url" in
        http://* | https://* ) ;;
        /*) doc_url="https://doc.arvados.org${doc_url}" ;;
        \#*) doc_url="https://doc.arvados.org/install/install-api-server.html${doc_url}" ;;
        *) doc_url="https://doc.arvados.org/install/${doc_url}" ;;
    esac
    cat >&2 <<EOF
NOTE: The $PACKAGE_NAME package was not configured completely because
$reason.
Please refer to the documentation for next steps:
  <$doc_url>

After you do that, resume $PACKAGE_NAME setup by running:
  $RESETUP_CMD
EOF
    exit "${exitcode:-20}"
}

setup_confdirs() {
    local confdir confgrp
    case "$WWW_OWNER" in
        "") confgrp=root ;;
        *) confgrp="$WWW_OWNER" ;;
    esac
    for confdir in "$@"; do
        if [ ! -d "$confdir" ]; then
            install -d -g "$confgrp" -m 0750 "$confdir"
        fi
    done
}

setup_conffile() {
    # Usage: setup_conffile CONFFILE_PATH [SOURCE_PATH]
    # Both paths are relative to RELEASE_CONFIG_PATH.
    # This function will try to safely ensure that a symbolic link for
    # the configuration file points from RELEASE_CONFIG_PATH to CONFIG_PATH.
    # If SOURCE_PATH is given, this function will try to install that file as
    # the configuration file in CONFIG_PATH, and return 1 if the file in
    # CONFIG_PATH is unmodified from the source.
    local conffile_relpath="$1"; shift
    local conffile_source="$1"
    local release_conffile="$RELEASE_CONFIG_PATH/$conffile_relpath"
    local etc_conffile="$CONFIG_PATH/$(basename "$conffile_relpath")"

    # Note that -h can return true and -e will return false simultaneously
    # when the target is a dangling symlink.  We're okay with that outcome,
    # so check -h first.
    if [ ! -h "$release_conffile" ]; then
        if [ ! -e "$release_conffile" ]; then
            ln -s "$etc_conffile" "$release_conffile"
        # If there's a config file in /var/www identical to the one in /etc,
        # overwrite it with a symlink after porting its permissions.
        elif cmp --quiet "$release_conffile" "$etc_conffile"; then
            local ownership="$(stat -c "%u:%g" "$release_conffile")"
            local owning_group="${ownership#*:}"
            if [ 0 != "$owning_group" ]; then
                chgrp "$owning_group" "$CONFIG_PATH" /etc/arvados
            fi
            chown "$ownership" "$etc_conffile"
            chmod --reference="$release_conffile" "$etc_conffile"
            ln --force -s "$etc_conffile" "$release_conffile"
        fi
    fi

    if [ -n "$conffile_source" ]; then
        if [ ! -e "$etc_conffile" ]; then
            install -g "$WWW_OWNER" -m 0640 \
                    "$RELEASE_CONFIG_PATH/$conffile_source" "$etc_conffile"
            return 1
        # Even if $etc_conffile already existed, it might be unmodified from
        # the source.  This is especially likely when a user installs, updates
        # database.yml, then reconfigures before they update application.yml.
        # Use cmp to be sure whether $etc_conffile is modified.
        elif cmp --quiet "$RELEASE_CONFIG_PATH/$conffile_source" "$etc_conffile"; then
            return 1
        fi
    fi
}

prepare_database() {
  # Prevent PostgreSQL from trying to page output
  unset PAGER
  DB_MIGRATE_STATUS=`"$BUNDLE" exec bin/rake db:migrate:status 2>&1 || true`
  if echo "$DB_MIGRATE_STATUS" | grep -qF 'Schema migrations table does not exist yet.'; then
      # The database exists, but the migrations table doesn't.
      run_and_report "Setting up database" "$BUNDLE" exec bin/rake db:schema:load db:seed
  elif echo "$DB_MIGRATE_STATUS" | grep -q '^database: '; then
      run_and_report "Running db:migrate" "$BUNDLE" exec bin/rake db:migrate
  elif echo "$DB_MIGRATE_STATUS" | grep -q 'database .* does not exist'; then
      run_and_report "Running db:setup" "$BUNDLE" exec bin/rake db:setup
  else
      # We don't have enough configuration to even check the database.
      return 1
  fi
}

case "$DISTRO_FAMILY" in
    debian) WWW_OWNER=www-data ;;
    rhel) WWW_OWNER="$(id --group --name nginx || true)" ;;
esac

# Before we do anything else, make sure some directories and files are in place
if [ ! -e $SHARED_PATH/log ]; then mkdir -p $SHARED_PATH/log; fi
if [ ! -e $RELEASE_PATH/tmp ]; then mkdir -p $RELEASE_PATH/tmp; fi
if [ ! -e $RELEASE_PATH/log ]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
if [ ! -e $SHARED_PATH/log/production.log ]; then touch $SHARED_PATH/log/production.log; fi

cd "$RELEASE_PATH"
export RAILS_ENV=production

run_and_report "Installing bundler" gem install --conservative --version '~> 2.4.0' bundler
ruby_minor_ver="$(ruby -e 'puts RUBY_VERSION.split(".")[..1].join(".")')"
BUNDLE="$(gem contents --version '~> 2.4.0' bundler | grep -E '/(bin|exe)/bundle$' | tail -n1)"
if ! [ -x "$BUNDLE" ]; then
    # Some distros (at least Ubuntu 24.04) append the Ruby version to the
    # executable name, but that isn't reflected in the output of
    # `gem contents`. Check for that version.
    BUNDLE="$BUNDLE$ruby_minor_ver"
    if ! [ -x "$BUNDLE" ]; then
        echo "Error: failed to find \`bundle\` command after installing bundler gem" >&2
        exit 11
    fi
fi

bundle_path="$SHARED_PATH/vendor_bundle"
run_and_report "Running bundle config set --local path $SHARED_PATH/vendor_bundle" \
               "$BUNDLE" config set --local path "$bundle_path"

# As of April 2024/Bundler 2.4, `bundle install` tends not to install gems
# which are already installed system-wide, which causes bundle activation to
# fail later. Prevent this by trying to pre-install all gems manually.
# `gem install` can fail if there are conflicts between gems installed by
# previous versions and gems installed by the current version. Ignore those
# errors; all that matters is that we get `bundle install` to succeed, and
# we check that next. <https://dev.arvados.org/issues/22647>
echo "Preinstalling bundle gems -- conflict errors are OK..."
find vendor/cache -maxdepth 1 -name '*.gem' -print0 |
    xargs -0r gem install --conservative --ignore-dependencies \
          --local --no-document --quiet \
          --install-dir="$bundle_path/ruby/$ruby_minor_ver.0" ||
    true
echo " done."
run_and_report "Running bundle install" "$BUNDLE" install --prefer-local --quiet
run_and_report "Verifying bundle is complete" "$BUNDLE" exec true

passenger="$("$BUNDLE" exec gem contents passenger | grep -E '/(bin|exe)/passenger$' | tail -n1)"
if ! [ -x "$passenger" ]; then
    echo "Error: failed to find \`passenger\` command after installing bundle" >&2
    exit 12
fi
"$BUNDLE" exec "$passenger-config" build-native-support
# `passenger-config install-standalone-runtime` downloads an agent, but at
# least with Passenger 6.0.23 (late 2024), that version tends to segfault.
# Compiling our own is safer.
"$BUNDLE" exec "$passenger-config" compile-agent --auto --optimize
"$BUNDLE" exec "$passenger-config" install-standalone-runtime --auto --brief

echo -n "Creating symlinks to configuration in $CONFIG_PATH ..."
setup_confdirs /etc/arvados "$CONFIG_PATH"
setup_conffile environments/production.rb environments/production.rb.example \
    || true
# Rails 5.2 does not tolerate dangling symlinks in the initializers
# directory, and this one can still be there, left over from a previous
# version of the API server package.
rm -f $RELEASE_PATH/config/initializers/omniauth.rb
echo "... done."

echo -n "Extending systemd unit configuration ..."
if [ -z "$WWW_OWNER" ]; then
    systemd_group="%N"
else
    systemd_group="$(systemd_quote "$WWW_OWNER")"
fi
install -d /lib/systemd/system/arvados-railsapi.service.d
# The 20 prefix is chosen so most user overrides should come after, which
# is what most admins will expect, but there's still space to put drop-ins
# earlier.
cat >/lib/systemd/system/arvados-railsapi.service.d/20-postinst.conf <<EOF
[Service]
ExecStartPre=+/bin/chgrp $systemd_group log tmp
ExecStartPre=+-/bin/chgrp $systemd_group \${PASSENGER_LOG_FILE}
ExecStart=
ExecStart=$(systemd_quote "$BUNDLE") exec $(systemd_quote "$passenger") start --daemonize --pid-file %t/%N/passenger.pid
ExecStop=
ExecStop=$(systemd_quote "$BUNDLE") exec $(systemd_quote "$passenger") stop --pid-file %t/%N/passenger.pid
ExecReload=
ExecReload=$(systemd_quote "$BUNDLE") exec $(systemd_quote "$passenger-config") reopen-logs
${WWW_OWNER:+SupplementaryGroups=$WWW_OWNER}
EOF
systemd_ctl daemon-reload
echo "... done."

# warn about config errors (deprecated/removed keys from
# previous version, etc)
if ! run_and_report "Checking configuration for completeness" "$BUNDLE" exec bin/rake config:check; then
    report_not_ready 21 "you must add required configuration settings to /etc/arvados/config.yml" "#update-config"
elif ! prepare_database; then
    report_not_ready 22 "database setup could not be completed"
else
    systemd_ctl try-restart arvados-railsapi.service
fi
