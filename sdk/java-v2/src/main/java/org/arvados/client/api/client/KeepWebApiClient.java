/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import okhttp3.HttpUrl;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;
import okhttp3.ResponseBody;

import org.arvados.client.config.ConfigProvider;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;

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

    public byte[] downloadPartial(String collectionUuid, String filePathName, long offset) throws IOException {
        Request.Builder builder = this.getRequestBuilder();
        if (offset > 0) {
            builder.addHeader("Range", "bytes=" + offset + "-");
        }
        Request request = builder.url(this.getUrlBuilder(collectionUuid, filePathName).build()).get().build();
        try (Response response = client.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Failed to download file: " + response);
            }
            try (ResponseBody body = response.body()) {
                if (body != null) {
                    return body.bytes();
                } else {
                    throw new IOException("Response body is null for request: " + request);
                }
            }
        }
    }

    public String delete(String collectionUuid, String filePathName) {
        Request request = getRequestBuilder()
                .url(getUrlBuilder(collectionUuid, filePathName).build())
                .delete()
                .build();

        return newCall(request);
    }

    public String upload(String collectionUuid, File file, ProgressListener progressListener) {
        RequestBody requestBody = new CountingFileRequestBody(file, progressListener);

        Request request = getRequestBuilder()
                .url(getUrlBuilder(collectionUuid, file.getName()).build())
                .put(requestBody)
                .build();
        return newCall(request);
    }

    public String upload(String collectionUuid, InputStream inputStream, String fileName, ProgressListener progressListener) {
        RequestBody requestBody = new CountingStreamRequestBody(inputStream, progressListener);

        Request request = getRequestBuilder()
                .url(getUrlBuilder(collectionUuid, fileName).build())
                .put(requestBody)
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
