# Install Arvados SDK into bcbio-nextgen Docker image.
#
# To build bcbio-nextgen:
#
# $ git clone https://github.com/chapmanb/bcbio-nextgen.git
# $ cd bcbio-nextgen
# $ docker build .
# $ docker tag <image> bcbio-nextgen
#

FROM bcbio-nextgen
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

# Set up system and Install dependencies
RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

# Install Arvados packages.
RUN /usr/local/share/bcbio-nextgen/anaconda/bin/pip install arvados-python-client

USER crunch
