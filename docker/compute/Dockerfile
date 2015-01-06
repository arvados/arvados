# Arvados compute node Docker container.

FROM arvados/slurm
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN apt-get update -qq
RUN apt-get install -qqy supervisor python-pip python-pyvcf python-gflags python-google-api-python-client python-virtualenv libattr1-dev libfuse-dev python-dev python-llfuse fuse crunchstat python-arvados-fuse cron dnsmasq

ADD fuse.conf /etc/fuse.conf
RUN chmod 644 /etc/fuse.conf

RUN /usr/local/rvm/bin/rvm-exec default gem install arvados-cli arvados

# Install Docker from the Arvados package repository (cf. arvados/base)
RUN apt-get install -qqy iptables ca-certificates lxc apt-transport-https docker.io

RUN addgroup --gid 4005 crunch && mkdir /home/crunch && useradd --uid 4005 --gid 4005 crunch && usermod crunch -G fuse,docker && chown crunch:crunch /home/crunch

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD generated/setup.sh /usr/local/bin/setup.sh
ADD wrapdocker /usr/local/bin/wrapdocker.sh

VOLUME /var/lib/docker
# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
