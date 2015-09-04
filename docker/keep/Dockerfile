# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

ADD generated/bin/keepstore /usr/local/bin/
ADD generated/bin/keepproxy /usr/local/bin/
ADD generated/run-keep /usr/local/bin/

ADD generated/keep_signing_secret /etc/

RUN mkdir /keep-data

# Start keep
CMD ["/usr/local/bin/run-keep"]
