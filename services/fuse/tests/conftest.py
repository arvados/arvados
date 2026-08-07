# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import multiprocessing

import pytest

# Tests were written when the default start method was fork, and still need it.
@pytest.fixture(scope='session', autouse=True)
def multiprocessing_start_method():
    multiprocessing.set_start_method('fork')
