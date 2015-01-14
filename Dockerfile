FROM arvados/base
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN apt-get update -qq
RUN apt-get install -qqy \
        apt-utils git curl procps apache2-mpm-worker \
        libcurl4-openssl-dev apache2-threaded-dev \
        libapr1-dev libaprutil1-dev

# Install apache configuration...

ADD apache2_vhost /etc/apache2/sites-available/arv-web
RUN \
  a2dissite default && \
  a2ensite arv-web && \
  a2enmod rewrite

ADD apache2_foreground.sh /etc/apache2/foreground.sh

CMD ["/etc/apache2/foreground.sh"]