FROM arvados/passenger
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

ADD apache2_foreground.sh /etc/apache2/foreground.sh

ADD apache2_vhost /etc/apache2/sites-available/arv-web
RUN \
  mkdir /var/run/apache2 && \
  a2dissite default && \
  a2ensite arv-web && \
  a2enmod rewrite

EXPOSE 80

CMD ["/etc/apache2/foreground.sh"]