# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import abc

class DownloaderBase(abc.ABC):
    def __init__(self):
        self.collection = None
        self.target = None
        self.name = None

    @abc.abstractmethod
    def head(self, url):
        ...

    @abc.abstractmethod
    def download(self, url, headers):
        ...
