case "$TARGET" in
    centos*)
        fpm_depends+=(glibc)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libc6)
        ;;
esac
