case "$TARGET" in
    debian* | ubuntu*)
        fpm_depends+=('libcurl4-gnutls-dev')
        ;;
    centos*)
        fpm_depends+=('libcurl')
        ;;
esac
