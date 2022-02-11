/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.HttpUrl;
import okhttp3.Request;
import org.arvados.client.config.ConfigProvider;

public class KeepWebApiClient extends BaseApiClient {

    public KeepWebApiClient(ConfigProvider config) {
        super(config);
    }

    public byte[] download(String collectionUuid, String filePathName) {
        Request request = getRequestBuilder()
                .url(getUrlBuilder(collectionUuid,filePathName).build())
                .get()
                .build();

        return newFileCall(request);
    }

    public String delete(String collectionUuid, String filePathName) {
        Request request = getRequestBuilder()
                .url(getUrlBuilder(collectionUuid, filePathName).build())
                .delete()
                .build();

        return newCall(request);
    }

    private HttpUrl.Builder getUrlBuilder(String collectionUuid, String filePathName) {
        return new HttpUrl.Builder()
                .scheme(config.getApiProtocol())
                .host(config.getKeepWebHost())
                .port(config.getKeepWebPort())
                .addPathSegment("c=" + collectionUuid)
                .addPathSegment(filePathName);
    }
}
