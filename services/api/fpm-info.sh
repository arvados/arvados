fpm_depends+=('git >= 1.7.10')

case "$TARGET" in
    centos*)
        fpm_depends+=(libcurl-devel postgresql-devel)
        ;;
    debian* | ubuntu*)
        fpm_depends+=(libcurl-ssl-dev libpq-dev)
        ;;
esac
