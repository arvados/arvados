FROM debian:8
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

# Install RVM
RUN apt-get update && apt-get -y install curl && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1

RUN echo "deb file:///arvados/packages/debian8/ /" >>/etc/apt/sources.list
