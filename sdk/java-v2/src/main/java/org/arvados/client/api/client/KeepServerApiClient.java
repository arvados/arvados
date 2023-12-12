/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.Request;
import okhttp3.RequestBody;
import org.arvados.client.api.client.CountingRequestBody.TransferData;
import org.arvados.client.common.Headers;
import org.arvados.client.config.ConfigProvider;
import org.slf4j.Logger;

import java.io.File;
import java.util.Map;

public class KeepServerApiClient extends BaseApiClient {

    private final Logger log = org.slf4j.LoggerFactory.getLogger(KeepServerApiClient.class);

    public KeepServerApiClient(ConfigProvider config) {
        super(config);
    }

    public String upload(String url, Map<String, String> headers, File body) {

        log.debug("Upload file {} to server location {}", body, url);

        final TransferData transferData = new TransferData(body.length());

        RequestBody requestBody =  new CountingFileRequestBody(body, transferData::updateTransferProgress);

        Request request = getRequestBuilder()
                .url(url)
                .addHeader(Headers.X_KEEP_DESIRED_REPLICAS, headers.get(Headers.X_KEEP_DESIRED_REPLICAS))
                .put(requestBody)
                .build();

        return newCall(request);
    }

    public byte[] download(String url) {

        Request request = getRequestBuilder()
                .url(url)
                .get()
                .build();

        return newFileCall(request);
    }
}
