FROM debian:8
RUN apt-get update

# preload (but don't install) packages arvados-boot might decide to install
RUN DEBIAN_FRONTEND=noninteractive apt-get -dy install --no-install-recommends ca-certificates locales nginx postgresql runit

RUN DEBIAN_FRONTEND=noninteractive apt-get -y install --no-install-recommends runit locales

RUN ["bash", "-c", "echo en_US.utf8 UTF-8 | tee -a /etc/locale.gen && locale-gen -a && \
    (echo LANG=en_US.UTF-8; echo LC_ALL=en_US.UTF-8) > /etc/default/locale"]

RUN DEBIAN_FRONTEND=noninteractive apt-get -y install --no-install-recommends ca-certificates locales nginx postgresql runit

CMD ["bash", "-c", "runsvdir /etc/sv & service postgresql start & arvados-server setup -init-vault && arvados-server setup -wait"]
