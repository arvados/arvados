# After you build this image, you should do a privileged run to make /dev/fuse.
# See the jobs-image recipe in docker/build_tools/Makefile.

FROM arvados/base
MAINTAINER Brett Smith <brett@curoverse.com>

# Install dependencies and set up system.
RUN /usr/bin/apt-get install -q -y python-dev python-llfuse python-pip \
      libio-socket-ssl-perl libjson-perl liburi-perl libwww-perl \
      fuse libattr1-dev libfuse-dev && \
    /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/sbin/adduser crunch fuse && \
    /usr/bin/install -d -o crunch -g crunch -m 0700 /tmp/crunch-job && \
    /bin/ln -s /proc/mounts /etc/mtab && \
    /bin/ln -s /usr/src/arvados /usr/local/src/arvados

# Install Arvados packages.
RUN find /usr/src/arvados/sdk -name '*.gem' -print0 | \
      xargs -0rn 1 gem install && \
    cd /usr/src/arvados/sdk/python && \
    python setup.py install
