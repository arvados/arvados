case "$TARGET" in
    centos*)
        fpm_depends+=(glibc)
        ;;
    debian8)
        fpm_depends+=(libc6 libgmp10)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libc6)
        ;;
esac
