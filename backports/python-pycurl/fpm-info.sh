case "$TARGET" in
    debian* | ubuntu*)
        fpm_depends+=('libcurl4-gnutls-dev | libcurl4-openssl-dev')
        ;;
    centos*)
        fpm_depends+=('libcurl')
        ;;
esac
