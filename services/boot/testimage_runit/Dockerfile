FROM debian:8
RUN apt-get update

# preload (but don't install) packages arvados-boot might decide to install
RUN DEBIAN_FRONTEND=noninteractive apt-get -dy install --no-install-recommends ca-certificates nginx runit

RUN DEBIAN_FRONTEND=noninteractive apt-get -y install --no-install-recommends runit

CMD ["bash", "-c", "coproc arvados-boot; runsvdir /etc/sv"]
