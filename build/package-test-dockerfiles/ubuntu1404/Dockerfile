# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:trusty
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies and RVM
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates python2.7-dev python3 python-setuptools python3-setuptools libcurl4-gnutls-dev curl git libattr1-dev libfuse-dev libpq-dev python-pip unzip binutils build-essential ca-certificates comerr-dev cpp cpp-4.8 dpkg-dev fakeroot g++ g++-4.8 gcc gcc-4.8 git-man krb5-locales krb5-multidev libalgorithm-diff-perl libalgorithm-diff-xs-perl libalgorithm-merge-perl libasan0 libasn1-8-heimdal libatomic1 libc-dev-bin libc6-dev libcloog-isl4 libcurl3 libcurl3-gnutls libdpkg-perl libedit2 liberror-perl libexpat1-dev libfakeroot libfile-fcntllock-perl libfuse2 libgcc-4.8-dev libgcrypt11-dev libglib2.0-0 libglib2.0-data libgmp10 libgnutls-dev libgnutlsxx27 libgomp1 libgpg-error-dev libgssapi-krb5-2 libgssapi3-heimdal libgssrpc4 libhcrypto4-heimdal libheimbase1-heimdal libheimntlm0-heimdal libhx509-5-heimdal libidn11 libidn11-dev libisl10 libitm1 libk5crypto3 libkadm5clnt-mit9 libkadm5srv-mit9 libkdb5-7 libkeyutils1 libkrb5-26-heimdal libkrb5-3 libkrb5-dev libkrb5support0 libldap-2.4-2 libldap2-dev libmpc3 libmpfr4 libp11-kit-dev libpcre3-dev libpcrecpp0 libpq5 libpython-stdlib libpython2.7 libpython2.7-dev libpython2.7-minimal libpython2.7-stdlib libquadmath0 libroken18-heimdal librtmp-dev librtmp0 libsasl2-2 libsasl2-modules libsasl2-modules-db libselinux1-dev libsepol1-dev libssl-dev libssl-doc libssl1.0.0 libstdc++-4.8-dev libtasn1-6-dev libtimedate-perl libtsan0 libwind0-heimdal libx11-6 libx11-data libxau6 libxcb1 libxdmcp6 libxext6 libxml2 libxmuu1 linux-libc-dev make manpages manpages-dev openssh-client openssl patch pkg-config python python-chardet python-chardet-whl python-colorama python-colorama-whl python-distlib python-distlib-whl python-html5lib python-html5lib-whl python-minimal python-pip-whl python-pkg-resources python-requests python-requests-whl python-setuptools-whl python-six python-six-whl python-urllib3 python-urllib3-whl python-wheel python2.7 python2.7-minimal python3-pkg-resources rsync sgml-base shared-mime-info xauth xml-core xz-utils libx11-6 libx11-data libxau6 libxcb1 libxdmcp6 libxext6 libxmuu1 openssh-client xauth  && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/ubuntu1404/ /" >>/etc/apt/sources.list
