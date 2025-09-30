# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import functools
import hashlib
import json
import logging
import operator
import os

import arvados.util

_FILTER_OPS = {
    '=': operator.eq,
    '>=': operator.ge,
    '>': operator.gt,
    '<=': operator.le,
    '<': operator.lt,
    '!=': operator.ne,
    '<>': operator.ne,
}

class DeferExecution:
    def __init__(self, fn):
        self._fn = fn

    def execute(self, *, num_retries=None):
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
        filepath = os.path.join(blockdir, arvados.KeepLocator(locator).md5sum)
        with open(filepath, "rb") as fr:
            return fr.read()

    def put(self, data, copies=2, num_retries=None, request_id=None, classes=None):
        md5 = hashlib.md5(data).hexdigest()
        locator = '%s+%d' % (md5, len(data))

        blockdir = os.path.join(self._basedir, locator[0:3])
        os.makedirs(blockdir, exist_ok=True)
        filepath = os.path.join(blockdir, md5)

        with open(os.path.join(filepath + '.tmp'), 'wb') as f:
            f.write(data)
        os.rename(os.path.join(filepath + '.tmp'),
                  os.path.join(filepath))
        return locator


def match_filter(fl, obj):
    key, op_key, val = fl
    try:
        op_func = _FILTER_OPS[op_key]
    except KeyError:
        raise NotImplementedError(f"unsupported filter operator {op_key}") from None
    else:
        return op_func(obj[key], val)


class StubArvadosResources:
    def __init__(self, basedir, resource_type):
        self._basedir = basedir
        self._resource_type = resource_type
        self._logger = logging.getLogger(f'arvados.stubapi.{resource_type}')

    @defer_execution
    def get(self, *, uuid=""):
        with open(os.path.join(self._basedir, uuid), "rt") as fr:
            return json.load(fr)

    @defer_execution
    def create(self, *, body=None, ensure_unique_name=None):
        if self._resource_type in body:
            body = body[self._resource_type]
        with open(os.path.join(self._basedir, body["uuid"]), "wt") as fw:
            json.dump(body, fw, indent=2)

        return body

    @defer_execution
    def update(self, *, uuid="", body=None):
        if self._resource_type in body:
            body = body[self._resource_type]
        with open(os.path.join(self._basedir, uuid), "rt") as fr:
            obj = json.load(fr)

        for k,v in body.items():
            obj[k] = v

        with open(os.path.join(self._basedir, uuid), "wt") as fw:
            json.dump(obj, fw, indent=2)

        return obj

    @defer_execution
    def list(self, *, filters=None, limit=None, count=None, order=None):
        items = []
        for dirent in os.scandir(self._basedir):
            if not arvados.util.uuid_pattern.match(dirent.name) or not dirent.is_file():
                continue

            with open(os.path.join(self._basedir, dirent.name), "rt") as fr:
                obj = json.load(fr)

            if all(match_filter(f, obj) for f in filters):
                items.append(obj)

        if order:
            if len(order) == 1:
                k1, r1 = order[0].split(' ')
                keycomp = lambda x: x[k1]
            elif len(order) == 2:
                k1, r1 = order[0].split(' ')
                k2, r2 = order[1].split(' ')
                if r1 != r2:
                    raise NotImplementedError("Can't have secondary sort column in opposite direction")
                keycomp = lambda x: (x[k1], x[k2])

            items.sort(key=keycomp, reverse=(r1=='desc'))

        if limit is not None:
            items = items[0:limit]

        return {
            "items": items,
            "items_available": len(items)
        }


class StubArvadosAPI:
    def __init__(self, basedir):
        self._basedir = basedir

        os.makedirs(os.path.join(self._basedir, "keep"), exist_ok=True)
        os.makedirs(os.path.join(self._basedir, "arvados/v1/collections"), exist_ok=True)
        os.makedirs(os.path.join(self._basedir, "arvados/v1/links"), exist_ok=True)
        os.makedirs(os.path.join(self._basedir, "arvados/v1/groups"), exist_ok=True)
        os.makedirs(os.path.join(self._basedir, "arvados/v1/workflows"), exist_ok=True)

        self.keep = StubKeepClient(os.path.join(self._basedir, "keep"))

    @classmethod
    def for_cwd(cls):
        return cls(os.getcwd())

    def collections(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/collections"), "collection")

    def links(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/links"), "link")

    def groups(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/groups"), "group")

    def workflows(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/workflows"), "workflow")
