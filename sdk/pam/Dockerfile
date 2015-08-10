# Manual integration test:
# 0. python setup.py sdist rotate --keep=1 --match .tar.gz
# 1. docker build -name arvados:pam_test .
# 2. docker run -it arvados:pam_test
# 3. container# edit /etc/pam.d/login  # set api host and shell VM name
# 4. container# useradd testusername
# 5. container# login                  # enter username and token

FROM debian:wheezy
RUN apt-get update
RUN apt-get -qy dist-upgrade
RUN apt-get -qy install python python-virtualenv libpam-python rsyslog
# Packages required by pycurl, ciso8601
RUN apt-get -qy install libcurl4-gnutls-dev python2.7-dev
RUN pip install --upgrade setuptools
RUN pip install python-pam
ADD dist /dist
RUN pip install /dist/arvados-pam-*.tar.gz
RUN DEBIAN_FRONTEND=noninteractive pam-auth-update arvados --remove unix
CMD rsyslogd & tail -F /var/log/auth.log & bash
