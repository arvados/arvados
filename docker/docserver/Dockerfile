# Arvados Documentation Docker container.

FROM arvados/base
maintainer Ward Vandewege <ward@clinicalfuture.com>

# Install packages
# Fix Locale
# Build static site
RUN /bin/mkdir -p /usr/src/arvados
ADD generated/doc.tar.gz /usr/src/arvados/
RUN \
    apt-get update ;\
    apt-get install -q -y curl procps apache2-mpm-worker ;\
    /bin/sed -ri 's/^baseurl: .*$/baseurl: /' /usr/src/arvados/doc/_config.yml ;\
    cd /usr/src/arvados/doc; LANG="en_US.UTF-8" LC_ALL="en_US.UTF-8" jekyll build

# Configure Apache
ADD apache2_vhost /etc/apache2/sites-available/doc
RUN \
  a2dissite default ;\
  a2ensite doc

ADD apache2_foreground.sh /etc/apache2/foreground.sh

# Start Apache
CMD ["/etc/apache2/foreground.sh"]
