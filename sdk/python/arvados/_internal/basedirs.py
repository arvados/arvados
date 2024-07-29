# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Base directories utility module

This module provides a set of classes useful to search and manipulate base
directory defined by systemd and the XDG specification. Most users will just
instantiate and use `BaseDirectories`.
"""

import dataclasses
import enum
import itertools
import logging
import os
import shlex
import stat

from pathlib import Path, PurePath
from typing import (
    Iterator,
    Mapping,
    Optional,
    Union,
)

logger = logging.getLogger('arvados')

@dataclasses.dataclass
class BaseDirectorySpec:
    """Parse base directories

    A BaseDirectorySpec defines all the environment variable keys and defaults
    related to a set of base directories (cache, config, state, etc.). It
    provides pure methods to parse environment settings into valid paths.
    """
    systemd_key: str
    xdg_home_key: str
    xdg_home_default: PurePath
    xdg_dirs_key: Optional[str] = None
    xdg_dirs_default: str = ''

    @staticmethod
    def _abspath_from_env(env: Mapping[str, str], key: str) -> Optional[Path]:
        try:
            path = Path(env[key])
        except (KeyError, ValueError):
            ok = False
        else:
            ok = path.is_absolute()
        return path if ok else None

    @staticmethod
    def _iter_abspaths(value: str) -> Iterator[Path]:
        for path_s in value.split(':'):
            path = Path(path_s)
            if path.is_absolute():
                yield path

    def iter_systemd(self, env: Mapping[str, str]) -> Iterator[Path]:
        return self._iter_abspaths(env.get(self.systemd_key, ''))

    def iter_xdg(self, env: Mapping[str, str], subdir: PurePath) -> Iterator[Path]:
        yield self.xdg_home(env, subdir)
        if self.xdg_dirs_key is not None:
            for path in self._iter_abspaths(env.get(self.xdg_dirs_key) or self.xdg_dirs_default):
                yield path / subdir

    def xdg_home(self, env: Mapping[str, str], subdir: PurePath) -> Path:
        return (
            self._abspath_from_env(env, self.xdg_home_key)
            or self.xdg_home_default_path(env)
        ) / subdir

    def xdg_home_default_path(self, env: Mapping[str, str]) -> Path:
        return (self._abspath_from_env(env, 'HOME') or Path.home()) / self.xdg_home_default

    def xdg_home_is_customized(self, env: Mapping[str, str]) -> bool:
        xdg_home = self._abspath_from_env(env, self.xdg_home_key)
        return xdg_home is not None and xdg_home != self.xdg_home_default_path(env)


class BaseDirectorySpecs(enum.Enum):
    """Base directory specifications

    This enum provides easy access to the standard base directory settings.
    """
    CACHE = BaseDirectorySpec(
        'CACHE_DIRECTORY',
        'XDG_CACHE_HOME',
        PurePath('.cache'),
    )
    CONFIG = BaseDirectorySpec(
        'CONFIGURATION_DIRECTORY',
        'XDG_CONFIG_HOME',
        PurePath('.config'),
        'XDG_CONFIG_DIRS',
        '/etc/xdg',
    )
    STATE = BaseDirectorySpec(
        'STATE_DIRECTORY',
        'XDG_STATE_HOME',
        PurePath('.local', 'state'),
    )


class BaseDirectories:
    """Resolve paths from a base directory spec

    Given a BaseDirectorySpec, this class provides stateful methods to find
    existing files and return the most-preferred directory for writing.
    """
    _STORE_MODE = stat.S_IFDIR | stat.S_IWUSR

    def __init__(
            self,
            spec: Union[BaseDirectorySpec, BaseDirectorySpecs, str],
            env: Mapping[str, str]=os.environ,
            xdg_subdir: Union[os.PathLike, str]='arvados',
    ) -> None:
        if isinstance(spec, str):
            spec = BaseDirectorySpecs[spec].value
        elif isinstance(spec, BaseDirectorySpecs):
            spec = spec.value
        self._spec = spec
        self._env = env
        self._xdg_subdir = PurePath(xdg_subdir)

    def search(self, name: str) -> Iterator[Path]:
        any_found = False
        for search_path in itertools.chain(
                self._spec.iter_systemd(self._env),
                self._spec.iter_xdg(self._env, self._xdg_subdir),
        ):
            path = search_path / name
            if path.exists():
                yield path
                any_found = True
        # The rest of this function is dedicated to warning the user if they
        # have a custom XDG_*_HOME value that prevented the search from
        # succeeding. This should be rare.
        if any_found or not self._spec.xdg_home_is_customized(self._env):
            return
        default_home = self._spec.xdg_home_default_path(self._env)
        default_path = Path(self._xdg_subdir / name)
        if not (default_home / default_path).exists():
            return
        if self._spec.xdg_dirs_key is None:
            suggest_key = self._spec.xdg_home_key
            suggest_value = default_home
        else:
            suggest_key = self._spec.xdg_dirs_key
            cur_value = self._env.get(suggest_key, '')
            value_sep = ':' if cur_value else ''
            suggest_value = f'{cur_value}{value_sep}{default_home}'
        logger.warning(
            "\
%s was not found under your configured $%s (%s), \
but does exist at the default location (%s) - \
consider running this program with the environment setting %s=%s\
",
            default_path,
            self._spec.xdg_home_key,
            self._spec.xdg_home(self._env, ''),
            default_home,
            suggest_key,
            shlex.quote(suggest_value),
        )

    def storage_path(
            self,
            subdir: Union[str, os.PathLike]=PurePath(),
            mode: int=0o700,
    ) -> Path:
        for path in self._spec.iter_systemd(self._env):
            try:
                mode = path.stat().st_mode
            except OSError:
                continue
            if (mode & self._STORE_MODE) == self._STORE_MODE:
                break
        else:
            path = self._spec.xdg_home(self._env, self._xdg_subdir)
        path /= subdir
        path.mkdir(parents=True, exist_ok=True, mode=mode)
        return path
