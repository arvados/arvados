FROM debian:wheezy
RUN apt-get update
RUN apt-get -qy dist-upgrade
RUN apt-get -qy install python python-virtualenv libpam-python rsyslog
# Packages required by pycurl, ciso8601
RUN apt-get -qy install libcurl4-gnutls-dev python2.7-dev
RUN pip install --upgrade setuptools
RUN pip install python-pam
ADD . /pam
WORKDIR /pam
RUN ./setup.py sdist
RUN pip install dist/arvados-pam-*.tar.gz
COPY examples/pam-auth-update_arvados /usr/share/pam-configs/arvados
RUN DEBIAN_FRONTEND=noninteractive pam-auth-update arvados --remove unix
CMD rsyslogd & tail -F /var/log/auth.log & bash
