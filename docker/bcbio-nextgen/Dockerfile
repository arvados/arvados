# Install Arvados SDK into bcbio-nextgen Docker image.
#
# To build bcbio-nextgen:
#
# $ git clone https://github.com/chapmanb/bcbio-nextgen.git
# $ cd bcbio-nextgen
# $ docker build
# $ docker tag <image> bcbio-nextgen
#

FROM bcbio-nextgen
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

USER root

# Install Ruby 2.1.0
RUN apt-get remove --quiet --assume-yes ruby && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1.0 && \
    /bin/mkdir -p /usr/src/arvados

ADD generated/arvados.tar.gz /usr/src/arvados/
ENV GEM_HOME /usr/local/rvm/gems/ruby-2.1.0
ENV GEM_PATH /usr/local/rvm/gems/ruby-2.1.0:/usr/local/rvm/gems/ruby-2.1.0@global
ENV PATH /usr/local/rvm/gems/ruby-2.1.0/bin:/usr/local/rvm/gems/ruby-2.1.0@global/bin:/usr/local/rvm/rubies/ruby-2.1.0/bin:/usr/local/rvm/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Install dependencies and set up system.
# The FUSE packages help ensure that we can install the Python SDK (arv-mount).
RUN /usr/bin/apt-get update && \
    /usr/bin/apt-get install --quiet --assume-yes python-dev python-llfuse python-pip \
      libio-socket-ssl-perl libjson-perl liburi-perl libwww-perl \
      fuse libattr1-dev libfuse-dev && \
    /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job && \
    /bin/ln -s /usr/src/arvados /usr/local/src/arvados

# Install Arvados packages.
RUN gem update --system && \
    find /usr/src/arvados/sdk -name '*.gem' -print0 | \
      xargs -0rn 1 gem install && \
    cd /usr/src/arvados/services/fuse && \
    python setup.py install && \
    cd /usr/src/arvados/sdk/python && \
    python setup.py install

USER crunch
