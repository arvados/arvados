# Arvados base image (wheezy+rvm+Arvados source) in Docker

# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# TODO(twp): parameterize variables via autoconf or similar.
ENV RUBY_VERSION_NUM   2.0.0

# Install prerequisite packages for Arvados
RUN apt-get update
RUN apt-get -q -y install -q -y apt-utils
RUN apt-get -q -y install git curl
RUN curl -L https://get.rvm.io | bash -s stable --ruby=${RUBY_VERSION_NUM}

# Set up RVM environment. These are just the env variables created by
# /usr/local/rvm/scripts/rvm, which can't be run from a non-login shell.
# TODO(twp): figure out a less awful solution.
ENV RUBY_VERSION ruby-2.0.0-p247
ENV MY_RUBY_HOME /usr/local/rvm/rubies/ruby-2.0.0-p247
ENV GEM_HOME /usr/local/rvm/gems/ruby-2.0.0-p247
ENV GEM_PATH /usr/local/rvm/gems/ruby-2.0.0-p247:/usr/local/rvm/gems/ruby-2.0.0-p247@global
ENV IRBRC /usr/local/rvm/rubies/ruby-2.0.0-p247/.irbrc
ENV rvm_path /usr/local/rvm
ENV rvm_prefix /usr/local
ENV rvm_env_string ruby-2.0.0-p247
ENV rvm_version 1.23.13 (stable)
ENV rvm_ruby_string ruby-2.0.0-p247
ENV PATH /usr/local/rvm/gems/ruby-2.0.0-p247/bin:/usr/local/rvm/gems/ruby-2.0.0-p247@global/bin:/usr/local/rvm/rubies/ruby-2.0.0-p247/bin:/usr/local/rvm/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Download Arvados source.
RUN git clone git://github.com/clinicalfuture/arvados.git /usr/src/arvados

