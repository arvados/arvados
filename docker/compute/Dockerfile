# Arvados compute node Docker container.

FROM arvados/slurm
MAINTAINER Ward Vandewege <ward@curoverse.com>

ADD apt.arvados.org.list /etc/apt/sources.list.d/
RUN apt-key adv --keyserver pgp.mit.edu --recv 1078ECD7

RUN apt-get update && apt-get -qqy install supervisor python-pip python-pyvcf python-gflags python-google-api-python-client python-virtualenv libattr1-dev libfuse-dev python-dev python-llfuse fuse crunchstat python-arvados-python-client python-arvados-fuse cron

ADD fuse.conf /etc/fuse.conf

RUN /usr/local/rvm/bin/rvm-exec default gem install arvados-cli arvados

# Install Docker from the Docker Inc. repository
RUN apt-get update -qq && apt-get install -qqy iptables ca-certificates lxc apt-transport-https
RUN echo deb https://get.docker.io/ubuntu docker main > /etc/apt/sources.list.d/docker.list
RUN apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9
RUN apt-get update -qq && apt-get install -qqy lxc-docker

RUN addgroup --gid 4005 crunch && mkdir /home/crunch && useradd --uid 4005 --gid 4005 crunch && usermod crunch -G fuse,docker && chown crunch:crunch /home/crunch

# Fix /dev/fuse permissions/ownership
RUN chmod 1660 /dev/fuse && chgrp fuse /dev/fuse

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD ssh.sh /usr/local/bin/ssh.sh
ADD generated/setup.sh /usr/local/bin/setup.sh
ADD wrapdocker /usr/local/bin/wrapdocker.sh

VOLUME /var/lib/docker
# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
