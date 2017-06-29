# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def application(environ, start_response):
    start_response('200 OK', [('Content-Type', 'text/plain')])
    return [b"hello world from python!\n"]
