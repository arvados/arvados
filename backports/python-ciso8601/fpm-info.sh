case "$TARGET" in
    centos*)
        fpm_depends+=(glibc)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libc6)
        ;;
esac

# FIXME: Remove this line after #6885 is done.
fpm_args+=(--iteration 2)
