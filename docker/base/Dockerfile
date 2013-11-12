# Arvados base image (wheezy+rvm+Arvados source) in Docker

# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install prerequisite packages for Arvados
#   * git, curl, rvm
#   * Arvados source code
#
RUN apt-get update && \
    apt-get -q -y install -q -y apt-utils && \
    apt-get -q -y install git curl && \
    git clone git://github.com/clinicalfuture/arvados.git /usr/src/arvados && \
    curl -L https://get.rvm.io | bash -s stable --ruby=2.0.0

# Set up RVM environment. These are just the env variables created by
# /usr/local/rvm/scripts/rvm, which can't be run from a non-login shell.
# TODO(twp): figure out a less awful solution.
ENV GEM_HOME /usr/local/rvm/gems/ruby-2.0.0-p247
ENV GEM_PATH /usr/local/rvm/gems/ruby-2.0.0-p247:/usr/local/rvm/gems/ruby-2.0.0-p247@global
ENV PATH /usr/local/rvm/gems/ruby-2.0.0-p247/bin:/usr/local/rvm/gems/ruby-2.0.0-p247@global/bin:/usr/local/rvm/rubies/ruby-2.0.0-p247/bin:/usr/local/rvm/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
