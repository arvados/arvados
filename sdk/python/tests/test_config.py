# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os

import pytest

from arvados import config as arv_config

class TestInitialize:
    @pytest.fixture(autouse=True)
    def setup(self, monkeypatch):
        arv_config._settings = None
        monkeypatch.delenv('ARVADOS_API_HOST', raising=False)
        monkeypatch.delenv('ARVADOS_API_TOKEN', raising=False)
        try:
            yield
        finally:
            arv_config._settings = None

    @pytest.fixture
    def tmp_settings(self, tmp_path):
        path = tmp_path / 'settings.conf'
        with path.open('w') as settings_file:
            print("ARVADOS_API_HOST=localhost", file=settings_file)
            print("ARVADOS_API_TOKEN=TestInitialize", file=settings_file)
        return path

    def test_static_path(self, tmp_settings):
        arv_config.initialize(tmp_settings)
        actual = arv_config.settings()
        assert actual['ARVADOS_API_HOST'] == 'localhost'
        assert actual['ARVADOS_API_TOKEN'] == 'TestInitialize'

    def test_search_path(self, tmp_settings):
        def search(filename):
            assert filename == tmp_settings.name
            yield tmp_settings
        arv_config.initialize(search)
        actual = arv_config.settings()
        assert actual['ARVADOS_API_HOST'] == 'localhost'
        assert actual['ARVADOS_API_TOKEN'] == 'TestInitialize'

    def test_default_search(self, tmp_settings, monkeypatch):
        monkeypatch.setenv('CONFIGURATION_DIRECTORY', str(tmp_settings.parent))
        monkeypatch.setenv('XDG_CONFIG_HOME', str(tmp_settings.parent))
        monkeypatch.delenv('XDG_CONFIG_DIRS', raising=False)
        actual = arv_config.settings()
        assert actual['ARVADOS_API_HOST'] == 'localhost'
        assert actual['ARVADOS_API_TOKEN'] == 'TestInitialize'

    def test_environ_override(self, monkeypatch):
        monkeypatch.setenv('ARVADOS_API_TOKEN', 'test_environ_override')
        arv_config.initialize('')
        actual = arv_config.settings()
        assert actual.get('ARVADOS_API_HOST') is None
        assert actual['ARVADOS_API_TOKEN'] == 'test_environ_override'
