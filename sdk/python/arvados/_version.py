# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import pkg_resources

__version__ = pkg_resources.require('arvados-python-client')[0].version
