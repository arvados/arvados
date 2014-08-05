FROM arvados/base
MAINTAINER Brett Smith <brett@curoverse.com>

# Install dependencies and set up system.
# The FUSE packages help ensure that we can install the Python SDK (arv-mount).
RUN /usr/bin/apt-get install -q -y python-dev python-llfuse python-pip \
      libio-socket-ssl-perl libjson-perl liburi-perl libwww-perl dtrx \
      fuse libattr1-dev libfuse-dev && \
    /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job && \
    /bin/ln -s /usr/src/arvados /usr/local/src/arvados

# Install Arvados packages.
RUN (find /usr/src/arvados/sdk -name '*.gem' -print0 | \
      xargs -0rn 1 /usr/local/rvm/bin/rvm-exec default gem install) && \
    cd /usr/src/arvados/services/fuse && \
    python setup.py install && \
    cd /usr/src/arvados/sdk/python && \
    python setup.py install

USER crunch
