#!/bin/sh
set -e
cd /var/www/arvados-workbench/current/

case "$TARGET" in
    debian*|ubuntu*)
        apt-get install -y nginx
        dpkg-reconfigure arvados-workbench
        ;;
    centos6)
        yum install --assumeyes httpd
        yum reinstall --assumeyes arvados-workbench
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac

/usr/local/rvm/bin/rvm-exec default bundle list >"$ARV_PACKAGES_DIR/arvados-workbench.gems"
