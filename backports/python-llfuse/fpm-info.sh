case "$TARGET" in
    centos6)
        build_depends+=('fuse-libs' 'fuse-devel')
        fpm_depends+=(glibc 'fuse-libs = 2.9.2-5' 'fuse = 2.9.2-5')
        ;;
    centos*)
        build_depends+=('fuse-devel')
        fpm_depends+=(glibc fuse-libs)
        ;;
    ubuntu1204)
        build_depends+=(libfuse2 libfuse-dev)
        fpm_depends+=(libc6 python-contextlib2 'libfuse2 = 2.9.2-5' 'fuse = 2.9.2-5')
        ;;
    debian* | ubuntu*)
        build_depends+=('libfuse-dev')
        fpm_depends+=(libc6 'libfuse2 > 2.9.0' 'fuse > 2.9.0')
        ;;
esac

# FIXME: Remove this line after #6885 is done.
fpm_args+=(--iteration 3)
