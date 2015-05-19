# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

RUN apt-get update -qq
RUN apt-get install -qqy ca-certificates

ADD generated/bin/keepproxy /usr/local/bin/
ADD generated/run-keepproxy /usr/local/bin/

# Start keep
CMD ["/usr/local/bin/run-keepproxy"]
