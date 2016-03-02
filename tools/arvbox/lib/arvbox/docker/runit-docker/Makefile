CFLAGS=-std=c99 -Wall -O2 -fPIC -D_POSIX_SOURCE -D_GNU_SOURCE
LDLIBS=-ldl

PROGNAME=runit-docker

all: $(PROGNAME).so

%.so: %.c
	gcc -shared $(CFLAGS) $(LDLIBS) -o $@ $^

install: runit-docker.so
	mkdir -p $(DESTDIR)/sbin
	mkdir -p $(DESTDIR)/lib
	install -m 755 $(PROGNAME) $(DESTDIR)/sbin/
	install -m 755 $(PROGNAME).so $(DESTDIR)/lib/

clean:
	$(RM) $(PROGNAME).so
