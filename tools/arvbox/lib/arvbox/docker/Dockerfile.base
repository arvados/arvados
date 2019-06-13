# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:9

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get -yq --no-install-recommends -o Acquire::Retries=6 install \
    postgresql-9.6 postgresql-contrib-9.6 git build-essential runit curl libpq-dev \
    libcurl4-openssl-dev libssl1.0-dev zlib1g-dev libpcre3-dev \
    openssh-server python-setuptools netcat-traditional \
    python-epydoc graphviz bzip2 less sudo virtualenv \
    libpython-dev fuse libfuse-dev python-pip python-yaml \
    pkg-config libattr1-dev python-llfuse python-pycurl \
    libwww-perl libio-socket-ssl-perl libcrypt-ssleay-perl \
    libjson-perl nginx gitolite3 lsof libreadline-dev \
    apt-transport-https ca-certificates \
    linkchecker python3-virtualenv python-virtualenv xvfb iceweasel \
    libgnutls28-dev python3-dev vim cadaver cython gnupg dirmngr \
    libsecret-1-dev r-base r-cran-testthat libxml2-dev pandoc \
    python3-setuptools python3-pip openjdk-8-jdk bsdmainutils && \
    apt-get clean

ENV RUBYVERSION_MINOR 2.3
ENV RUBYVERSION 2.3.5

# Install Ruby from source
RUN cd /tmp && \
 curl -f http://cache.ruby-lang.org/pub/ruby/${RUBYVERSION_MINOR}/ruby-${RUBYVERSION}.tar.gz | tar -xzf - && \
 cd ruby-${RUBYVERSION} && \
 ./configure --disable-install-doc && \
 make && \
 make install && \
 cd /tmp && \
 rm -rf ruby-${RUBYVERSION}

ENV GEM_HOME /var/lib/gems
ENV GEM_PATH /var/lib/gems
ENV PATH $PATH:/var/lib/gems/bin

ENV GOVERSION 1.11.5

# Install golang binary
RUN curl -f http://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz | \
    tar -C /usr/local -xzf -

ENV PATH ${PATH}:/usr/local/go/bin

VOLUME /var/lib/docker
VOLUME /var/log/nginx
VOLUME /etc/ssl/private

ADD 58118E89F3A912897C070ADBF76221572C52609D.asc /tmp/
RUN apt-key add --no-tty /tmp/58118E89F3A912897C070ADBF76221572C52609D.asc && \
    rm -f /tmp/58118E89F3A912897C070ADBF76221572C52609D.asc

RUN mkdir -p /etc/apt/sources.list.d && \
    echo deb https://apt.dockerproject.org/repo debian-stretch main > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get -yq --no-install-recommends install docker-engine=17.05.0~ce-0~debian-stretch && \
    apt-get clean

RUN rm -rf /var/lib/postgresql && mkdir -p /var/lib/postgresql

ENV PJSVERSION=1.9.8
# bitbucket is the origin, but downloads fail sometimes, so use our own mirror instead.
#ENV PJSURL=https://bitbucket.org/ariya/phantomjs/downloads/phantomjs-${PJSVERSION}-linux-x86_64.tar.bz2
ENV PJSURL=http://cache.arvados.org/phantomjs-${PJSVERSION}-linux-x86_64.tar.bz2

RUN set -e && \
 curl -L -f ${PJSURL} | tar -C /usr/local -xjf - && \
 ln -s ../phantomjs-${PJSVERSION}-linux-x86_64/bin/phantomjs /usr/local/bin

ENV GDVERSION=v0.23.0
ENV GDURL=https://github.com/mozilla/geckodriver/releases/download/$GDVERSION/geckodriver-$GDVERSION-linux64.tar.gz
RUN set -e && curl -L -f ${GDURL} | tar -C /usr/local/bin -xzf - geckodriver

RUN pip install -U setuptools

ENV NODEVERSION v8.15.1

# Install nodejs binary
RUN curl -L -f https://nodejs.org/dist/${NODEVERSION}/node-${NODEVERSION}-linux-x64.tar.xz | tar -C /usr/local -xJf - && \
    ln -s ../node-${NODEVERSION}-linux-x64/bin/node ../node-${NODEVERSION}-linux-x64/bin/npm /usr/local/bin

ENV GRADLEVERSION 5.3.1

RUN cd /tmp && \
    curl -L -O https://services.gradle.org/distributions/gradle-${GRADLEVERSION}-bin.zip && \
    unzip gradle-${GRADLEVERSION}-bin.zip -d /usr/local && \
    ln -s ../gradle-${GRADLEVERSION}/bin/gradle /usr/local/bin && \
    rm gradle-${GRADLEVERSION}-bin.zip

# Set UTF-8 locale
RUN echo en_US.UTF-8 UTF-8 > /etc/locale.gen && locale-gen
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

ARG arvados_version
RUN echo arvados_version is git commit $arvados_version

ADD fuse.conf /etc/

ADD crunch-setup.sh gitolite.rc \
    keep-setup.sh common.sh createusers.sh \
    logger runsu.sh waitforpostgres.sh \
    yml_override.py api-setup.sh \
    go-setup.sh devenv.sh \
    /usr/local/lib/arvbox/

ADD runit /etc/runit

# Start the supervisor.
ENV SVDIR /etc/service
STOPSIGNAL SIGINT
CMD ["/sbin/runit"]
