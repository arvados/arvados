# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN echo 'deb http://apt.arvados.org/ wheezy main' > /etc/apt/sources.list.d/apt.arvados.org.list

RUN /usr/bin/apt-key adv --keyserver pgp.mit.edu --recv 1078ECD7

RUN /usr/bin/apt-get update

RUN /usr/bin/apt-get install keep

# Start keep
CMD ["/usr/bin/keep"]
