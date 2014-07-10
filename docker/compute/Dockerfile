# Arvados compute node Docker container.

FROM arvados/slurm
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN apt-get update && apt-get -q -y install supervisor

RUN /usr/local/rvm/bin/rvm-exec default gem install arvados-cli arvados

# Supervisor.
ADD supervisor.conf /etc/supervisor/conf.d/arvados.conf
ADD ssh.sh /usr/local/bin/ssh.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
