# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import datetime
import logging
import time
import urllib.parse

import arvados
import arvados.collection

import boto3
import boto3.s3.transfer

from .downloaderbase import DownloaderBase
from .to_keep_util import (Response, url_to_keep, generic_check_cached_url)

logger = logging.getLogger('arvados.s3_import')


class _Downloader(DownloaderBase):
    def __init__(self, apiclient, botoclient):
        super().__init__()
        self.apiclient = apiclient
        self.botoclient = botoclient
        self.headresult = None

    def head(self, url):
        self.parsedurl = urllib.parse.urlparse(url)

        extraArgs = {}
        versionId = urllib.parse.parse_qs(self.parsedurl.query).get("versionId", [False])[0]
        if versionId:
            extraArgs["VersionId"] = versionId
            extraArgs["ResponseCacheControl"] = "immutable"
        response = self.botoclient.head_object(
            Bucket=self.parsedurl.netloc,
            Key=self.parsedurl.path.lstrip('/'),
            **extraArgs
        )
        return Response(response['ResponseMetadata']['HTTPStatusCode'],
                        {k.title(): v for k,v in response['ResponseMetadata']['HTTPHeaders'].items()})

    def download(self, url, headers):
        self.collection = arvados.collection.Collection(api_client=self.apiclient)

        self.count = 0
        self.start = time.time()
        self.checkpoint = self.start
        self.contentlength = None
        self.target = None

        self.parsedurl = urllib.parse.urlparse(url)
        extraArgs = {}
        versionId = urllib.parse.parse_qs(self.parsedurl.query).get("versionId", [None])[0]
        if versionId:
            extraArgs["VersionId"] = versionId

        self.name = self.parsedurl.path.split("/")[-1]
        self.target = self.collection.open(self.name, "wb")

        objectMeta = self.head(url)
        self.contentlength = int(objectMeta.headers["Content-Length"])

        self.botoclient.download_fileobj(
            Bucket=self.parsedurl.netloc,
            Key=self.parsedurl.path.lstrip('/'),
            Fileobj=self.target,
            ExtraArgs=extraArgs,
            Callback=self.data_received,
            Config=boto3.s3.transfer.TransferConfig(
                multipart_threshold=64*1024*1024,
                multipart_chunksize=64*1024*1024,
                use_threads=False,
            ))

        return objectMeta

    def data_received(self, count):
        self.count += count

        loopnow = time.time()
        if (loopnow - self.checkpoint) < 20:
            return

        bps = self.count / (loopnow - self.start)
        if self.contentlength is not None:
            logger.info("%2.1f%% complete, %6.2f MiB/s, %1.0f seconds left",
                        ((self.count * 100) / self.contentlength),
                        (bps / (1024.0*1024.0)),
                        ((self.contentlength-self.count) // bps))
        else:
            logger.info("%d downloaded, %6.2f MiB/s", self.count, (bps / (1024.0*1024.0)))
        self.checkpoint = loopnow

def get_botoclient(botosession, unsigned_requests):
    if unsigned_requests:
        from botocore import UNSIGNED
        from botocore.config import Config
        return botosession.client('s3', config=Config(signature_version=UNSIGNED))
    else:
        return botosession.client('s3')


def check_cached_url(api, botosession, project_uuid, url, etags,
                     utcnow=datetime.datetime.utcnow,
                     prefer_cached_downloads=False,
                     unsigned_requests=False):

    return generic_check_cached_url(api, _Downloader(api, get_botoclient(botosession, unsigned_requests)),
                            project_uuid, url, etags,
                            utcnow=utcnow,
                            prefer_cached_downloads=prefer_cached_downloads)

def s3_to_keep(api, botosession, project_uuid, url,
               utcnow=datetime.datetime.utcnow,
               prefer_cached_downloads=False,
               unsigned_requests=False):
    """Download a file over S3 and upload it to keep, with HTTP headers as metadata.

    Because simple S3 object fetches are just HTTP underneath, we can
    reuse most of the HTTP downloading infrastructure.
    """

    return url_to_keep(api, _Downloader(api, get_botoclient(botosession, unsigned_requests)),
                       project_uuid, url,
                       utcnow=utcnow,
                       prefer_cached_downloads=prefer_cached_downloads)
