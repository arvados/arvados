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

from .to_keep_util import (Response, url_to_keep, check_cached_url as generic_check_cached_url)

logger = logging.getLogger('arvados.s3_import')


class _Downloader:
    # Wait up to 60 seconds for connection
    # How long it can be in "low bandwidth" state before it gives up
    # Low bandwidth threshold is 32 KiB/s
    DOWNLOADER_TIMEOUT = (60, 300, 32768)

    def __init__(self, apiclient, botoclient):
        self.target = None
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

def check_cached_url(api, project_uuid, url, etags,
                     utcnow=datetime.datetime.utcnow,
                     prefer_cached_downloads=False):
    return generic_check_cached_url(api, _Downloader(api, boto3.client('s3')),
                            project_uuid, url, etags,
                            utcnow=utcnow,
                            prefer_cached_downloads=prefer_cached_downloads)


def s3_to_keep(api, project_uuid, url,
               utcnow=datetime.datetime.utcnow,
               prefer_cached_downloads=False):
    """Download a file over S3 and upload it to keep, with HTTP headers as metadata.

    Because simple S3 object fetches are just HTTP underneath, we can
    reuse most of the HTTP downloading infrastucture.
    """

    return url_to_keep(api, _Downloader(api, boto3.client('s3')),
                       project_uuid, url,
                       utcnow=utcnow,
                       prefer_cached_downloads=prefer_cached_downloads)
