# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import json
import functools

class DeferExecution:
    def __init__(self, fn):
        self._fn = fn

    def execute(self):
        return self._fn()

def defer_execution(f):
    @functools.wraps(f)
    def wrapper(*args, **kwds):
        return DeferExecution(functools.partial(f, *args, **kwds))
    return wrapper

class StubKeepClient:
    def __init__(self, basedir):
        self._basedir = basedir

    def get(self, locator):
        blockdir = os.path.join(self._basedir, locator[0:3])
        filepath = os.path.join(blockdir, locator)
        with open(filepath, "rb") as fr:
            return fr.read()

class StubArvadosResources:
    def __init__(self, basedir):
        self._basedir = basedir

    @defer_execution
    def get(self, *, uuid=""):
        with open(os.path.join(self._basedir, uuid), "rt") as fr:
            return json.load(fr)

    @defer_execution
    def list(self, *, filters=None):
        pass

class StubArvadosAPI:
    def __init__(self, basedir):
        self._basedir = basedir
        self.keep = StubKeepClient(os.path.join(self._basedir, "keep"))

    def collections(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/collections"))
