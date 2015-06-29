# Slurm node Docker container.

FROM arvados/base
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN apt-get update -q
RUN apt-get install -qy slurm-llnl munge

ADD munge.key /etc/munge/
RUN chown munge:munge /etc/munge/munge.key && chmod 600 /etc/munge/munge.key
ADD generated/slurm.conf /etc/slurm-llnl/

