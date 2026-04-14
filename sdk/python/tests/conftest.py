# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import pytest

from . import run_test_server

@pytest.fixture
def reset_test_server_db():
    """pytest fixture wrapper for run_test_server.reset()"""
    try:
        yield
    finally:
        run_test_server.reset()
