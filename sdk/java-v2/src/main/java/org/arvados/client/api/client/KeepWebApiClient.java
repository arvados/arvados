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

    public InputStream get(String collectionUuid, String filePathName, long start, Long end) throws IOException {
        Request.Builder builder = this.getRequestBuilder();
        String rangeValue = "bytes=" + start + "-";
        if (end != null) {
            rangeValue += end;
        }
        builder.addHeader("Range", rangeValue);
        Request request = builder.url(this.getUrlBuilder(collectionUuid, filePathName).build()).get().build();
        Response response = client.newCall(request).execute();
        if (!response.isSuccessful()) {
            response.close();
            throw new IOException("Failed to download file: " + response);
        }
        ResponseBody body = response.body();
        if (body == null) {
            response.close();
            throw new IOException("Response body is null for request: " + request);
        }
        return body.byteStream();
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
