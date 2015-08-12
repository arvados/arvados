# These tests assume you have a real API server running on the docker host.
#
# Build the test container:
#   First, replace 3000 below with your api server's port number if necessary.
#   host$ python setup.py sdist rotate --keep=1 --match .tar.gz
#   host$ docker build --tag=arvados/pam_test .
#
# Automated integration test:
#   host$ docker run -it --add-host zzzzz.arvadosapi.com:"$(hostname -I |awk '{print $1}')" arvados/pam_test
# You should see "=== OK ===", followed by a Perl stack trace due to a
# yet-unidentified pam_python.so bug.
#
# Manual integration test:
#   host$ docker run -it --add-host zzzzz.arvadosapi.com:"$(hostname -I |awk '{print $1}')" arvados/pam_test bash -c 'rsyslogd & tail -F /var/log/auth.log & sleep 1 & bash'
#   container# login
#   login: active
#   Arvados API token: 3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi
# You should now be logged in to the "active" shell account in the
# container. You should also see arvados_pam log entries in
# /var/log/auth.log (and in your terminal, thanks to "tail -F").

FROM debian:wheezy
RUN apt-get update
RUN apt-get -qy dist-upgrade
RUN apt-get -qy install python python-virtualenv libpam-python rsyslog
# Packages required by pycurl, ciso8601
RUN apt-get -qy install libcurl4-gnutls-dev python2.7-dev

# for jessie (which also has other snags)
# RUN apt-get -qy install python-pip libgnutls28-dev

RUN pip install --upgrade setuptools
RUN pip install python-pam
ADD dist /dist
RUN pip install /dist/arvados-pam-*.tar.gz

# Configure and enable the module (hopefully vendor packages will offer a neater way)
RUN perl -pi -e 's{api.example}{zzzzz.arvadosapi.com:3000}; s{shell\.example}{testvm2.shell insecure};' /usr/share/pam-configs/arvados
RUN DEBIAN_FRONTEND=noninteractive pam-auth-update arvados --remove unix

# Add a user account matching the fixture
RUN useradd -ms /bin/bash active

# Test with python (SIGSEGV during tests)
#ADD . /pam
#WORKDIR /pam
#CMD rsyslogd & tail -F /var/log/auth.log & python setup.py test --test-suite integration_tests

# Test with perl (SIGSEGV when program exits)
RUN apt-get install -qy libauthen-pam-perl
ADD tests/integration_test.pl /integration_test.pl
CMD rsyslogd & tail -F /var/log/auth.log & sleep 1 && /integration_test.pl
