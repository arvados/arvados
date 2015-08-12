case "$TARGET" in
    debian* | ubuntu*)
        fpm_depends+=('libcurl4-gnutls-dev' 'libyaml-dev')
        ;;
    centos*)
        fpm_depends+=('libcurl' 'libyaml')
        ;;
esac
