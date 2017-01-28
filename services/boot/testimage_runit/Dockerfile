FROM debian:8
RUN apt-get update

RUN DEBIAN_FRONTEND=noninteractive apt-get -y install --no-install-recommends runit
RUN mkdir /etc/sv/arvados-boot && ln -s /usr/bin/arvados-boot /etc/sv/arvados-boot/run

# preload (but don't install) packages arvados-boot might decide to install
RUN DEBIAN_FRONTEND=noninteractive apt-get -dy install ca-certificates

CMD ["sh", "-c", "runsvdir /etc/sv"]
