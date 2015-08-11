# Manual integration test:
# 0. python setup.py sdist rotate --keep=1 --match .tar.gz
# 1. replace 53015 below with your api server's port number
# 2. docker build -name arvados:pam_test .
# 3. docker run -it --add-host zzzzz.arvadosapi.com:"$(hostname -I |awk '{print $1}')" arvados:pam_test
# 4. container# useradd testusername
# 5. container# login                  # enter username and token

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
RUN perl -pi -e 's{api.example}{zzzzz.arvadosapi.com:53015}; s{shell\.example}{testvm2.shell insecure};' /usr/share/pam-configs/arvados
RUN DEBIAN_FRONTEND=noninteractive pam-auth-update arvados --remove unix

# Add a user account matching the fixture
RUN useradd -ms /bin/bash active

# Test with python (SIGSEGV during tests)
#ADD . /pam
#WORKDIR /pam
#CMD rsyslogd & tail -F /var/log/auth.log & python setup.py test

# Test with perl (SIGSEGV when program exits)
RUN apt-get install -qy libauthen-pam-perl
ADD tests/integration_test.pl /integration_test.pl
CMD rsyslogd & tail -F /var/log/auth.log & sleep 1 && /integration_test.pl
