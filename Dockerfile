FROM arvados/base
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN apt-get update -qq
RUN apt-get install -qqy \
        apt-utils git curl procps apache2-mpm-worker \
        libcurl4-openssl-dev apache2-threaded-dev \
        libapr1-dev libaprutil1-dev

RUN cd /usr/src/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle exec passenger-install-apache2-module --auto --languages ruby,python

RUN cd /usr/src/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle exec passenger-install-apache2-module --snippet > /etc/apache2/conf.d/passenger

ADD apache2_foreground.sh /etc/apache2/foreground.sh

ADD apache2_vhost /etc/apache2/sites-available/arv-web
RUN \
  mkdir /var/run/apache2 && \
  a2dissite default && \
  a2ensite arv-web && \
  a2enmod rewrite

EXPOSE 80

CMD ["/etc/apache2/foreground.sh"]