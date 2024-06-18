#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This code runs after package variable definitions and step2.sh.

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
# Default documentation URL. This can be set to a more specific URL.
NOT_READY_DOC_URL="https://doc.arvados.org/install/install-api-server.html"

report_web_service_warning() {
    local warning="$1"; shift
    cat >&2 <<EOF

WARNING: $warning.

To override, set the WEB_SERVICE environment variable to the name of the service
hosting the Rails server.

After you do that, resume $PACKAGE_NAME setup by running:
  $RESETUP_CMD
EOF
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

setup_confdirs() {
    for confdir in "$@"; do
        if [ ! -d "$confdir" ]; then
            install -d -g "$WWW_OWNER" -m 0750 "$confdir"
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
  DB_MIGRATE_STATUS=`bin/rake db:migrate:status 2>&1 || true`
  if echo "$DB_MIGRATE_STATUS" | grep -qF 'Schema migrations table does not exist yet.'; then
      # The database exists, but the migrations table doesn't.
      run_and_report "Setting up database" bin/rake \
                     "$RAILSPKG_DATABASE_LOAD_TASK" db:seed
  elif echo "$DB_MIGRATE_STATUS" | grep -q '^database: '; then
      run_and_report "Running db:migrate" \
                     bin/rake db:migrate
  elif echo "$DB_MIGRATE_STATUS" | grep -q 'database .* does not exist'; then
      run_and_report "Running db:setup" bin/rake db:setup
  else
      # We don't have enough configuration to even check the database.
      return 1
  fi
}

configure_version() {
  if [ -n "$WEB_SERVICE" ]; then
      SERVICE_MANAGER=$(guess_service_manager)
  elif WEB_SERVICE=$(list_services_systemd | grep -E '^(nginx|httpd)'); then
      SERVICE_MANAGER=systemd
  elif WEB_SERVICE=$(list_services_service \
                         | grep -Eo '\b(nginx|httpd)[^[:space:]]*'); then
      SERVICE_MANAGER=service
  fi

  if [ -z "$WEB_SERVICE" ]; then
    report_web_service_warning "Web service (Nginx or Apache) not found"
  elif [ "$WEB_SERVICE" != "$(echo "$WEB_SERVICE" | head -n 1)" ]; then
    WEB_SERVICE=$(echo "$WEB_SERVICE" | head -n 1)
    report_web_service_warning \
        "Multiple web services found.  Choosing the first one ($WEB_SERVICE)"
  fi

  case "$DISTRO_FAMILY" in
      debian) WWW_OWNER=www-data ;;
      rhel) case "$WEB_SERVICE" in
                httpd*) WWW_OWNER=apache ;;
                nginx*) WWW_OWNER=nginx ;;
            esac
            ;;
  esac

  # Before we do anything else, make sure some directories and files are in place
  if [ ! -e $SHARED_PATH/log ]; then mkdir -p $SHARED_PATH/log; fi
  if [ ! -e $RELEASE_PATH/tmp ]; then mkdir -p $RELEASE_PATH/tmp; fi
  if [ ! -e $RELEASE_PATH/log ]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
  if [ ! -e $SHARED_PATH/log/production.log ]; then touch $SHARED_PATH/log/production.log; fi

  cd "$RELEASE_PATH"
  export RAILS_ENV=production

  run_and_report "Installing bundler" gem install --conservative --version '~> 2.4.0' bundler
  local ruby_minor_ver="$(ruby -e 'puts RUBY_VERSION.split(".")[..1].join(".")')"
  local bundle="$(gem contents --version '~> 2.4.0' bundler | grep -E '/(bin|exe)/bundle$' | tail -n1)"
  if ! [ -x "$bundle" ]; then
      # Some distros (at least Ubuntu 24.04) append the Ruby version to the
      # executable name, but that isn't reflected in the output of
      # `gem contents`. Check for that version.
      bundle="$bundle$ruby_minor_ver"
      if ! [ -x "$bundle" ]; then
          echo "Error: failed to find \`bundle\` command after installing bundler gem" >&2
          return 1
      fi
  fi

  local bundle_path="$SHARED_PATH/vendor_bundle"
  run_and_report "Running bundle config set --local path $SHARED_PATH/vendor_bundle" \
                 "$bundle" config set --local path "$bundle_path"

  # As of April 2024/Bundler 2.4, `bundle install` tends not to install gems
  # which are already installed system-wide, which causes bundle activation to
  # fail later. Work around this by installing all gems manually.
  find vendor/cache -maxdepth 1 -name '*.gem' -print0 \
      | run_and_report "Installing bundle gems" xargs -0r \
                       gem install --conservative --ignore-dependencies --local --quiet \
                       --install-dir="$bundle_path/ruby/$ruby_minor_ver.0"
  run_and_report "Running bundle install" "$bundle" install --prefer-local --quiet
  run_and_report "Verifying bundle is complete" "$bundle" exec true

  if [ -z "$WWW_OWNER" ]; then
    NOT_READY_REASON="there is no web service account to own Arvados configuration"
    NOT_READY_DOC_URL="https://doc.arvados.org/install/nginx.html"
  else
    cat <<EOF

Assumption: $WEB_SERVICE is configured to serve Rails from
            $RELEASE_PATH
Assumption: $WEB_SERVICE and passenger run as $WWW_OWNER

EOF

    echo -n "Creating symlinks to configuration in $CONFIG_PATH ..."
    setup_confdirs /etc/arvados "$CONFIG_PATH"
    setup_conffile environments/production.rb environments/production.rb.example \
        || true
    setup_extra_conffiles
    echo "... done."

    echo -n "Ensuring directory and file permissions ..."
    # Ensure correct ownership of a few files
    chown "$WWW_OWNER:" $RELEASE_PATH/config/environment.rb
    chown "$WWW_OWNER:" $RELEASE_PATH/config.ru
    chown "$WWW_OWNER:" $RELEASE_PATH/Gemfile.lock
    chown -R "$WWW_OWNER:" $SHARED_PATH/log
    # Make sure postgres doesn't try to use a pager.
    export PAGER=
    case "$RAILSPKG_DATABASE_LOAD_TASK" in
        # db:structure:load was deprecated in Rails 6.1 and shouldn't be used.
        db:schema:load | db:structure:load)
            chown "$WWW_OWNER:" $RELEASE_PATH/db/schema.rb || true
            chown "$WWW_OWNER:" $RELEASE_PATH/db/structure.sql || true
            ;;
    esac
    chmod 644 $SHARED_PATH/log/*
    echo "... done."
  fi

  if [ -n "$NOT_READY_REASON" ]; then
      :
  # warn about config errors (deprecated/removed keys from
  # previous version, etc)
  elif ! run_and_report "Checking configuration for completeness" bin/rake config:check; then
      NOT_READY_REASON="you must add required configuration settings to /etc/arvados/config.yml"
      NOT_READY_DOC_URL="https://doc.arvados.org/install/install-api-server.html#update-config"
  elif [ -z "$RAILSPKG_DATABASE_LOAD_TASK" ]; then
      :
  elif ! prepare_database; then
      NOT_READY_REASON="database setup could not be completed"
  fi

  if [ -n "$WWW_OWNER" ]; then
    chown -R "$WWW_OWNER:" $RELEASE_PATH/tmp
    chmod -R 2775 $RELEASE_PATH/tmp
  fi

  if [ -z "$NOT_READY_REASON" ] && [ -n "$SERVICE_MANAGER" ]; then
      service_command "$SERVICE_MANAGER" restart "$WEB_SERVICE"
  fi
}

configure_version
if [ -n "$NOT_READY_REASON" ]; then
    cat >&2 <<EOF
NOTE: The $PACKAGE_NAME package was not configured completely because
$NOT_READY_REASON.
Please refer to the documentaion for next steps:
  <$NOT_READY_DOC_URL>

After you do that, resume $PACKAGE_NAME setup by running:
  $RESETUP_CMD
EOF
fi
