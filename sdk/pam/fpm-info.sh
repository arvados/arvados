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
        fpm_args+=('--deb-recommends=system-log-daemon')
        ;;
esac
