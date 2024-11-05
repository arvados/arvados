# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import re

import pytest

from arvados import _internal

class TestDeprecated:
    @staticmethod
    @_internal.deprecated('TestVersion', 'arvados.noop')
    def noop_func():
        """Do nothing

        This function returns None.
        """

    @pytest.mark.parametrize('pattern', [
        r'^Do nothing$',
        r'^ *.. WARNING:: Deprecated$',
        r' removed in Arvados TestVersion\.',
        r' Prefer arvados\.noop\b',
        r'^ *This function returns None\.$',
    ])
    def test_docstring(self, pattern):
        assert re.search(pattern, self.noop_func.__doc__, re.MULTILINE) is not None

    def test_deprecation_warning(self):
        with pytest.warns(DeprecationWarning) as check:
            self.noop_func()
        actual = str(check[0].message)
        assert ' removed in Arvados TestVersion.' in actual
        assert ' Prefer arvados.noop ' in actual
