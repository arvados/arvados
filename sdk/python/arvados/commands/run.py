# Copyright (C) The Arvados Authors. All rights reserved.
# Copyright (C) 2018 Genome Research Ltd.
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from __future__ import print_function
from __future__ import absolute_import
from builtins import range
from past.builtins import basestring
from builtins import object
import arvados
import arvados.commands.ws as ws
import argparse
import json
import re
import os
import stat
from . import put
import time
import subprocess
import logging
import sys
import errno
import arvados.commands._util as arv_cmd
import arvados.collection
import arvados.config as config

from arvados._version import __version__

logger = logging.getLogger('arvados.arv-run')
logger.setLevel(logging.INFO)

class ArvFile(object):
    def __init__(self, prefix, fn):
        self.prefix = prefix
        self.fn = fn

    def __hash__(self):
        return (self.prefix+self.fn).__hash__()

    def __eq__(self, other):
        return (self.prefix == other.prefix) and (self.fn == other.fn)

class UploadFile(ArvFile):
    pass

# Determine if a file is in a collection, and return a tuple consisting of the
# portable data hash and the path relative to the root of the collection.
# Return None if the path isn't with an arv-mount collection or there was is error.
def is_in_collection(root, branch):
    try:
        if root == "/":
            return (None, None)
        fn = os.path.join(root, ".arvados#collection")
        if os.path.exists(fn):
            with file(fn, 'r') as f:
                c = json.load(f)
            return (c["portable_data_hash"], branch)
        else:
            sp = os.path.split(root)
            return is_in_collection(sp[0], os.path.join(sp[1], branch))
    except (IOError, OSError):
        return (None, None)

# Determine the project to place the output of this command by searching upward
# for arv-mount psuedofile indicating the project.  If the cwd isn't within
# an arv-mount project or there is an error, return current_user.
def determine_project(root, current_user):
    try:
        if root == "/":
            return current_user
        fn = os.path.join(root, ".arvados#project")
        if os.path.exists(fn):
            with file(fn, 'r') as f:
                c = json.load(f)
            if 'writable_by' in c and current_user in c['writable_by']:
                return c["uuid"]
            else:
                return current_user
        else:
            sp = os.path.split(root)
            return determine_project(sp[0], current_user)
    except (IOError, OSError):
        return current_user

# Determine if string corresponds to a file, and if that file is part of a
# arv-mounted collection or only local to the machine.  Returns one of
# ArvFile() (file already exists in a collection), UploadFile() (file needs to
# be uploaded to a collection), or simply returns prefix+fn (which yields the
# original parameter string).
def statfile(prefix, fn, fnPattern="$(file %s/%s)", dirPattern="$(dir %s/%s/)", raiseOSError=False):
    absfn = os.path.abspath(fn)
    try:
        st = os.stat(absfn)
        sp = os.path.split(absfn)
        (pdh, branch) = is_in_collection(sp[0], sp[1])
        if pdh:
            if stat.S_ISREG(st.st_mode):
                return ArvFile(prefix, fnPattern % (pdh, branch))
            elif stat.S_ISDIR(st.st_mode):
                return ArvFile(prefix, dirPattern % (pdh, branch))
            else:
                raise Exception("%s is not a regular file or directory" % absfn)
        else:
            # trim leading '/' for path prefix test later
            return UploadFile(prefix, absfn[1:])
    except OSError as e:
        if e.errno == errno.ENOENT and not raiseOSError:
            pass
        else:
            raise

    return prefix+fn

def write_file(collection, pathprefix, fn, flush=False):
    with open(os.path.join(pathprefix, fn), "rb") as src:
        dst = collection.open(fn, "wb")
        r = src.read(1024*128)
        while r:
            dst.write(r)
            r = src.read(1024*128)
        dst.close(flush=flush)

def uploadfiles(files, api, dry_run=False, num_retries=0,
                project=None,
                fnPattern="$(file %s/%s)",
                name=None,
                collection=None,
                packed=True):
    # Find the smallest path prefix that includes all the files that need to be uploaded.
    # This starts at the root and iteratively removes common parent directory prefixes
    # until all file paths no longer have a common parent.
    if files:
        n = True
        pathprefix = "/"
        while n:
            pathstep = None
            for c in files:
                if pathstep is None:
                    sp = c.fn.split('/')
                    if len(sp) < 2:
                        # no parent directories left
                        n = False
                        break
                    # path step takes next directory
                    pathstep = sp[0] + "/"
                else:
                    # check if pathstep is common prefix for all files
                    if not c.fn.startswith(pathstep):
                        n = False
                        break
            if n:
                # pathstep is common parent directory for all files, so remove the prefix
                # from each path
                pathprefix += pathstep
                for c in files:
                    c.fn = c.fn[len(pathstep):]

        logger.info("Upload local files: \"%s\"", '" "'.join([c.fn for c in files]))

    if dry_run:
        logger.info("$(input) is %s", pathprefix.rstrip('/'))
        pdh = "$(input)"
    else:
        files = sorted(files, key=lambda x: x.fn)
        if collection is None:
            collection = arvados.collection.Collection(api_client=api, num_retries=num_retries)
        prev = ""
        for f in files:
            localpath = os.path.join(pathprefix, f.fn)
            if prev and localpath.startswith(prev+"/"):
                # If this path is inside an already uploaded subdirectory,
                # don't redundantly re-upload it.
                # e.g. we uploaded /tmp/foo and the next file is /tmp/foo/bar
                # skip it because it starts with "/tmp/foo/"
                continue
            prev = localpath
            if os.path.isfile(localpath):
                write_file(collection, pathprefix, f.fn, not packed)
            elif os.path.isdir(localpath):
                for root, dirs, iterfiles in os.walk(localpath):
                    root = root[len(pathprefix):]
                    for src in iterfiles:
                        write_file(collection, pathprefix, os.path.join(root, src), not packed)

        pdh = None
        if len(collection) > 0:
            # non-empty collection
            filters = [["portable_data_hash", "=", collection.portable_data_hash()]]
            name_pdh = "%s (%s)" % (name, collection.portable_data_hash())
            if name:
                filters.append(["name", "=", name_pdh])
            if project:
                filters.append(["owner_uuid", "=", project])

            # do the list / create in a loop with up to 2 tries as we are using `ensure_unique_name=False`
            # and there is a potential race with other workflows that may have created the collection
            # between when we list it and find it does not exist and when we attempt to create it.
            tries = 2
            while pdh is None and tries > 0:
                exists = api.collections().list(filters=filters, limit=1).execute(num_retries=num_retries)

                if exists["items"]:
                    item = exists["items"][0]
                    pdh = item["portable_data_hash"]
                    logger.info("Using collection %s (%s)", pdh, item["uuid"])
                else:
                    try:
                        collection.save_new(name=name_pdh, owner_uuid=project, ensure_unique_name=False)
                        pdh = collection.portable_data_hash()
                        logger.info("Uploaded to %s (%s)", pdh, collection.manifest_locator())
                    except arvados.errors.ApiError as ae:
                        tries -= 1
            if pdh is None:
                # Something weird going on here, probably a collection
                # with a conflicting name but wrong PDH.  We won't
                # able to reuse it but we still need to save our
                # collection, so so save it with unique name.
                logger.info("Name conflict on '%s', existing collection has an unexpected portable data hash", name_pdh)
                collection.save_new(name=name_pdh, owner_uuid=project, ensure_unique_name=True)
                pdh = collection.portable_data_hash()
                logger.info("Uploaded to %s (%s)", pdh, collection.manifest_locator())
        else:
            # empty collection
            pdh = collection.portable_data_hash()
            assert (pdh == config.EMPTY_BLOCK_LOCATOR), "Empty collection portable_data_hash did not have expected locator, was %s" % pdh
            logger.debug("Using empty collection %s", pdh)

    for c in files:
        c.keepref = "%s/%s" % (pdh, c.fn)
        c.fn = fnPattern % (pdh, c.fn)


def main(arguments=None):
    raise Exception("Legacy arv-run removed.")

if __name__ == '__main__':
    main()
