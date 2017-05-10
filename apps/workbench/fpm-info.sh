case "$TARGET" in
    centos*)
        fpm_depends+=(git)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(git g++)
        ;;
esac
