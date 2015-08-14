FROM debian:jessie
MAINTAINER Ward Vandewege <ward@curoverse.com>

# Install dependencies and set up system.
RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y python2.7-dev python3 python-setuptools python3-setuptools libcurl4-gnutls-dev curl git procps libattr1-dev libfuse-dev libgnutls28-dev libpq-dev

# Install RVM
RUN gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm

# Install golang binary
ADD generated/golang-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

ENV WORKSPACE /arvados
CMD ["/usr/local/rvm/bin/rvm-exec", "default", "bash", "/jenkins/run-build-packages.sh", "--target", "debian8"]
