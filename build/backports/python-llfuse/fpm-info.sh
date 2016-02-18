case "$TARGET" in
    centos*)
        build_depends+=('fuse-devel')
        fpm_depends+=(glibc fuse-libs)
        ;;
    debian* | ubuntu*)
        build_depends+=('libfuse-dev')
        fpm_depends+=(libc6 libfuse2)
        ;;
esac

# FIXME: Remove this line after #6885 is done.
fpm_args+=(--iteration 2)

# FIXME: Remove once support for llfuse 0.42+ is in place
fpm_args+=(-v 0.41.1)
