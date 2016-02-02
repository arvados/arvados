case "$TARGET" in
    debian* | ubuntu*)
        # FIXME: Remove once support for llfuse 0.42+ is in place
        fpm_args+=(--deb-ignore-iteration-in-dependencies)
        ;;
esac
