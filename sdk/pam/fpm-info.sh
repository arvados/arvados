case "$TARGET" in
    debian* | ubuntu*)
        fpm_depends+=('libpam-python')
        ;;
    *)
        echo >&2 "ERROR: $PACKAGE: pam_python.so dependency unavailable in $TARGET."
        return 1
        ;;
esac

case "$FORMAT" in
    deb)
        fpm_args+=('--deb-recommends=rsyslog')
        ;;
esac

fpm_args+=('--config-files=examples/pam-auth-update_arvados')
