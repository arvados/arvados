#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This code runs after package variable definitions and step2.sh.

set -e

DATABASE_READY=1
APPLICATION_READY=1

if [ -s "$HOME/.rvm/scripts/rvm" ] || [ -s "/usr/local/rvm/scripts/rvm" ]; then
    COMMAND_PREFIX="/usr/local/rvm/bin/rvm-exec default"
else
    COMMAND_PREFIX=
fi

report_not_ready() {
    local ready_flag="$1"; shift
    local config_file="$1"; shift
    if [ "1" != "$ready_flag" ]; then cat >&2 <<EOF

PLEASE NOTE:

The $PACKAGE_NAME package was not configured completely because
$config_file needs some tweaking.
Please refer to the documentation at
<$DOC_URL> for more details.

When $(basename "$config_file") has been modified,
reconfigure or reinstall this package.

EOF
    fi
}

report_web_service_warning() {
    local warning="$1"; shift
    cat >&2 <<EOF

WARNING: $warning.

To override, set the WEB_SERVICE environment variable to the name of the service
hosting the Rails server.

For Debian-based systems, then reconfigure this package with dpkg-reconfigure.

For RPM-based systems, then reinstall this package.

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
  DB_MIGRATE_STATUS=`$COMMAND_PREFIX bundle exec rake db:migrate:status 2>&1 || true`
  if echo "$DB_MIGRATE_STATUS" | grep -qF 'Schema migrations table does not exist yet.'; then
      # The database exists, but the migrations table doesn't.
      run_and_report "Setting up database" $COMMAND_PREFIX bundle exec \
                     rake "$RAILSPKG_DATABASE_LOAD_TASK" db:seed
  elif echo "$DB_MIGRATE_STATUS" | grep -q '^database: '; then
      run_and_report "Running db:migrate" \
                     $COMMAND_PREFIX bundle exec rake db:migrate
  elif echo "$DB_MIGRATE_STATUS" | grep -q 'database .* does not exist'; then
      if ! run_and_report "Running db:setup" \
           $COMMAND_PREFIX bundle exec rake db:setup 2>/dev/null; then
          echo "Warning: unable to set up database." >&2
          DATABASE_READY=0
      fi
  else
    echo "Warning: Database is not ready to set up. Skipping database setup." >&2
    DATABASE_READY=0
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

  if [ -e /etc/redhat-release ]; then
      # Recognize any service that starts with "nginx"; e.g., nginx16.
      if [ "$WEB_SERVICE" != "${WEB_SERVICE#nginx}" ]; then
        WWW_OWNER=nginx
      else
        WWW_OWNER=apache
      fi
  else
      # Assume we're on a Debian-based system for now.
      # Both Apache and Nginx run as www-data by default.
      WWW_OWNER=www-data
  fi

  echo
  echo "Assumption: $WEB_SERVICE is configured to serve Rails from"
  echo "            $RELEASE_PATH"
  echo "Assumption: $WEB_SERVICE and passenger run as $WWW_OWNER"
  echo

  echo -n "Creating symlinks to configuration in $CONFIG_PATH ..."
  setup_confdirs /etc/arvados "$CONFIG_PATH"
  setup_conffile environments/production.rb environments/production.rb.example \
      || true
  setup_extra_conffiles
  echo "... done."

  # Before we do anything else, make sure some directories and files are in place
  if [ ! -e $SHARED_PATH/log ]; then mkdir -p $SHARED_PATH/log; fi
  if [ ! -e $RELEASE_PATH/tmp ]; then mkdir -p $RELEASE_PATH/tmp; fi
  if [ ! -e $RELEASE_PATH/log ]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
  if [ ! -e $SHARED_PATH/log/production.log ]; then touch $SHARED_PATH/log/production.log; fi

  cd "$RELEASE_PATH"
  export RAILS_ENV=production

  if ! $COMMAND_PREFIX bundle --version >/dev/null; then
      run_and_report "Installing bundler" $COMMAND_PREFIX gem install bundler --version 1.17.3
  fi

  run_and_report "Running bundle install" \
      $COMMAND_PREFIX bundle install --path $SHARED_PATH/vendor_bundle --local --quiet

  echo -n "Ensuring directory and file permissions ..."
  # Ensure correct ownership of a few files
  chown "$WWW_OWNER:" $RELEASE_PATH/config/environment.rb
  chown "$WWW_OWNER:" $RELEASE_PATH/config.ru
  chown "$WWW_OWNER:" $RELEASE_PATH/Gemfile.lock
  chown -R "$WWW_OWNER:" $RELEASE_PATH/tmp || true
  chown -R "$WWW_OWNER:" $SHARED_PATH/log
  # Make sure postgres doesn't try to use a pager.
  export PAGER=
  case "$RAILSPKG_DATABASE_LOAD_TASK" in
      db:schema:load) chown "$WWW_OWNER:" $RELEASE_PATH/db/schema.rb ;;
      db:structure:load) chown "$WWW_OWNER:" $RELEASE_PATH/db/structure.sql ;;
  esac
  chmod 644 $SHARED_PATH/log/*
  chmod -R 2775 $RELEASE_PATH/tmp || true
  echo "... done."

  if [ -n "$RAILSPKG_DATABASE_LOAD_TASK" ]; then
      prepare_database
  fi

  if [ -e /etc/arvados/config.yml ]; then
      # warn about config errors (deprecated/removed keys from
      # previous version, etc)
      run_and_report "Checking configuration for completeness" \
                     $COMMAND_PREFIX bundle exec rake config:check || true
  fi

  chown -R "$WWW_OWNER:" $RELEASE_PATH/tmp

  setup_before_nginx_restart

  if [ -n "$SERVICE_MANAGER" ]; then
      service_command "$SERVICE_MANAGER" restart "$WEB_SERVICE"
  fi
}

if [ "$1" = configure ]; then
  # This is a debian-based system
  configure_version
elif [ "$1" = "0" ] || [ "$1" = "1" ] || [ "$1" = "2" ]; then
  # This is an rpm-based system
  configure_version
fi

report_not_ready "$APPLICATION_READY" "/etc/arvados/config.yml"
