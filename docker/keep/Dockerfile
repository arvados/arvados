# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN echo 'deb http://apt.arvados.org/ wheezy main' > /etc/apt/sources.list.d/apt.arvados.org.list

RUN /usr/bin/apt-key adv --keyserver pgp.mit.edu --recv 1078ECD7

RUN /usr/bin/apt-get update

RUN /usr/bin/apt-get install keep

ADD generated/run-keep /usr/local/bin/
RUN chmod +x /usr/local/bin/run-keep

ADD generated/keep_signing_secret /etc/

# Start keep
CMD ["/usr/local/bin/run-keep"]
