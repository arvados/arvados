# Arvados Documentation Docker container.

FROM arvados/base
maintainer Ward Vandewege <ward@clinicalfuture.com>

env DEBIAN_FRONTEND noninteractive

# Install packages
# Fix Locale
# Build static site
# Update Arvados source
RUN \
    apt-get update ;\
    apt-get install -q -y curl procps apache2-mpm-worker locales ;\
    /bin/sed -ri 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen ;\
    /usr/sbin/locale-gen ;\
    gem install jekyll RedCloth ;\
    /bin/sed -ri 's/^baseurl: .*$/baseurl: /' /usr/src/arvados/doc/_config.yml ;\
    cd /usr/src/arvados/doc; LANG="en_US.UTF-8" LC_ALL="en_US.UTF-8" jekyll build ;\
    cd /usr/src/arvados ;\
    git pull

# Configure Apache
ADD apache2_vhost /etc/apache2/sites-available/doc
RUN \
  a2dissite default ;\
  a2ensite doc

# Finally, start Apache
env APACHE_RUN_USER    www-data
env APACHE_RUN_GROUP   www-data
env APACHE_PID_FILE    /var/run/apache2.pid
env APACHE_RUN_DIR     /var/run/apache2
env APACHE_LOCK_DIR    /var/lock/apache2
env APACHE_LOG_DIR     /var/log/apache2
env LANG               C

cmd ["/usr/sbin/apache2", "-D", "FOREGROUND"]
