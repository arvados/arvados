
# Based on Debian Wheezy
FROM arvados/debian:wheezy
maintainer Ward Vandewege <ward@clinicalfuture.com>

run echo "alias v='ls -laF'" >> /root/.bashrc

# TODO(twp): parameterize variables via autoconf or similar.
env RUBY_VERSION_NUM   2.0.0

# Need to set the mysql passwords before installing mysql-server
env DEBIAN_FRONTEND noninteractive

run apt-get update
run apt-get install -y vim

#RUN apt-get upgrade -y

# Install prerequisite packages for Arvados
run apt-get install -q -y apt-utils
run apt-get install -q -y git curl procps apache2-mpm-worker locales
run curl -L https://get.rvm.io | bash -s stable --ruby=${RUBY_VERSION_NUM}

# Set up RVM environment. These are just the env variables created by
# /usr/local/rvm/scripts/rvm, which can't be run from a non-login shell.
# TODO(twp): figure out a less awful solution.
env RUBY_VERSION ruby-2.0.0-p247
env MY_RUBY_HOME /usr/local/rvm/rubies/ruby-2.0.0-p247
env GEM_HOME /usr/local/rvm/gems/ruby-2.0.0-p247
env GEM_PATH /usr/local/rvm/gems/ruby-2.0.0-p247:/usr/local/rvm/gems/ruby-2.0.0-p247@global
env IRBRC /usr/local/rvm/rubies/ruby-2.0.0-p247/.irbrc
env rvm_path /usr/local/rvm
env rvm_prefix /usr/local
env rvm_env_string ruby-2.0.0-p247
env rvm_version 1.23.13 (stable)
env rvm_ruby_string ruby-2.0.0-p247
env PATH /usr/local/rvm/gems/ruby-2.0.0-p247/bin:/usr/local/rvm/gems/ruby-2.0.0-p247@global/bin:/usr/local/rvm/rubies/ruby-2.0.0-p247/bin:/usr/local/rvm/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Fix Locale
run apt-get install -q -y locales

run /bin/sed -ri 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
run /usr/sbin/locale-gen

# Install Jekyll
run gem install jekyll RedCloth

# Download Arvados source.
run git clone git://github.com/clinicalfuture/arvados.git /usr/src/arvados

# Fix baseurl for doc server
run /bin/sed -ri 's/^baseurl: .*$/baseurl: /' /usr/src/arvados/doc/_config.yml

# Build doc site
run cd /usr/src/arvados/doc; LANG="en_US.UTF-8" LC_ALL="en_US.UTF-8" jekyll build

# Configure Apache
add doc.arvados.org /etc/apache2/sites-available/
run ln -s /etc/apache2/sites-available/doc.arvados.org /etc/apache2/sites-enabled/
run rm -f /etc/apache2/sites-enabled/000-default

# Finally, start Apache
env APACHE_RUN_USER    www-data
env APACHE_RUN_GROUP   www-data
env APACHE_PID_FILE    /var/run/apache2.pid
env APACHE_RUN_DIR     /var/run/apache2
env APACHE_LOCK_DIR    /var/lock/apache2
env APACHE_LOG_DIR     /var/log/apache2
env LANG               C

cmd ["/usr/sbin/apache2", "-D", "FOREGROUND"]
