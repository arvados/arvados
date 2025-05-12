# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados filters for Ansible"""

import dataclasses
import ipaddress
import operator
import socket
import typing as t
import urllib.parse

from collections import abc

Config = abc.Mapping[str, t.Any]

class FilterModule:
    """Export functions as Jinja filters to Ansible"""
    _FILTERS_MAP: t.Dict[str, abc.Callable] = {}

    @classmethod
    def register(cls, func: abc.Callable) -> abc.Callable:
        cls._FILTERS_MAP[func.__name__] = func
        return func

    def filters(self) -> abc.Mapping[str, abc.Callable]:
        return self._FILTERS_MAP


@dataclasses.dataclass
class ListenAddress:
    """Parse and query an Arvados service's listen address"""
    address: str
    """The address a service should listen on. May be an IP address or hostname."""
    port: int
    """The port a service should listen on. May be 0 to be assigned a port."""

    GLOBAL_ADDR = ipaddress.ip_address('1.1.1.1')
    LOOPBACK_ADDR = ipaddress.ip_address('127.0.0.1')

    @classmethod
    def parse(cls, s: str) -> 'ListenAddress':
        """Parse a ListenAddress from a URL string"""
        parts = urllib.parse.urlparse(s)
        address = parts.hostname
        if address is None:
            raise ValueError(f"no address or hostname in {s!r}")
        port = parts.port
        if port is None:
            try:
                port = socket.getservbyname(parts.scheme)
            except (OSError, TypeError):
                raise ValueError(f"no port or known scheme in {s!r}")
        return cls(address, port)

    def sort_score(self) -> int:
        """Return a sort key for this address

        Used to choose the single "best" listen address in configuration.
        Returns an arbitrary integer that represents the priority of this
        address: smaller numbers means higher priority. The current order is:

        1. the "all addresses" zero address
        2. global addresses
        3. non-loopback addresses
        4. any address
        """
        try:
            addr = ipaddress.ip_address(self.address)
        except ValueError:
            # The address is a hostname. Synthesize an IP address for scoring.
            addr = self.LOOPBACK_ADDR if self.address == 'localhost' else self.GLOBAL_ADDR
        if int(addr) == 0:
            return 0
        elif addr.is_global:
            return 1
        elif not addr.is_loopback:
            return 2
        else:
            return 255

    def __str__(self) -> str:
        return f'{self.address}:{self.port}'


@FilterModule.register
def external_addr(svc_config: Config) -> ListenAddress:
    """Parse and return a listen address from a service's ExternalURL

    Pass in an Arvados service configuration like
    `arvados_cluster.Services.RailsAPI`. This function parses and returns the
    service's ExternalURL.
    """
    try:
        url = svc_config['ExternalURL']
    except KeyError:
        raise ValueError("no ExternalURL defined in service configuration")
    else:
        return ListenAddress.parse(url)


@FilterModule.register
def listen_addrs(svc_config: Config) -> abc.Iterator[ListenAddress]:
    """Iterate all listen addresses for an Arvados service

    Pass in an Arvados service configuration like
    `arvados_cluster.Services.RailsAPI`. This function iterates all valid
    listen addresses in the configuration.
    """
    for url, url_config in svc_config.get('InternalURLs', {}).items():
        listen_url = url_config.get('ListenURL', url)
        try:
            addr = ListenAddress.parse(listen_url)
        except ValueError:
            pass
        else:
            yield addr
    try:
        addr
    except NameError:
        raise ValueError("no valid ListenURLs in service configuration") from None


@FilterModule.register
def listen_addr(svc_config: Config) -> ListenAddress:
    """Return a single listen address for an Arvados service

    Pass in an Arvados service configuration like
    `arvados_cluster.Services.RailsAPI`. This function finds and returns the
    most preferred address to listen on.
    """
    try:
        return min(listen_addrs(svc_config), key=operator.methodcaller('sort_score'))
    except ValueError:
        raise ValueError("no listen URLs defined in service configuration") from None


@FilterModule.register
def internal_addrs(svc_config: Config) -> ListenAddress:
    """Iterate listen addresses from a service's InternalURLs

    Pass in an Arvados service configuration like
    `arvados_cluster.Services.RailsAPI`. This function parses and returns the
    service's InternalURLs.
    """
    for url in svc_config.get('InternalURLs', ()):
        try:
            addr = ListenAddress.parse(url)
        except ValueError:
            pass
        else:
            yield addr
    try:
        addr
    except NameError:
        raise ValueError("no valid InternalURLs in service configuration") from None
