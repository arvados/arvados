#!/bin/bash

case $TARGET in
     centos7)
         # fpm incorrectly transforms the dependency name in this case.
         fpm_depends+=(python-backports-ssl_match_hostname)
         fpm_args+=(--python-disable-dependency backports.ssl-match-hostname)
     ;;
esac
