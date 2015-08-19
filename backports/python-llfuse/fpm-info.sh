case "$TARGET" in
    centos*)
        build_depends+=('fuse-devel')
        fpm_depends+=('fuse')
        ;;
    debian* | ubuntu*)
        build_depends+=('libfuse-dev')
        fpm_depends+=('fuse')
        ;;
esac
