/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.arvados.client.exception.ArvadosApiException;
import org.arvados.client.api.client.factory.OkHttpClientFactory;
import org.arvados.client.api.model.ApiError;
import org.arvados.client.config.ConfigProvider;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import okhttp3.ResponseBody;
import org.slf4j.Logger;

import java.io.IOException;
import java.io.UnsupportedEncodingException;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.Objects;
import java.util.concurrent.TimeUnit;

abstract class BaseApiClient {

    static final ObjectMapper MAPPER = new ObjectMapper().findAndRegisterModules();

    final OkHttpClient client;
    final ConfigProvider config;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(BaseApiClient.class);

    BaseApiClient(ConfigProvider config) {
        this.config = config;
        this.client = OkHttpClientFactory.INSTANCE.create(config.isApiHostInsecure())
	    .newBuilder()
	    .connectTimeout(config.getConnectTimeout(), TimeUnit.MILLISECONDS)
	    .readTimeout(config.getReadTimeout(), TimeUnit.MILLISECONDS)
	    .writeTimeout(config.getWriteTimeout(), TimeUnit.MILLISECONDS)
	    .build();
    }

    Request.Builder getRequestBuilder() {
        return new Request.Builder()
                .addHeader("authorization", String.format("OAuth2 %s", config.getApiToken()))
                .addHeader("cache-control", "no-cache");
    }

    String newCall(Request request) {
        return (String) getResponseBody(request, body -> body.string().trim());
    }

    byte[] newFileCall(Request request) {
        return (byte[]) getResponseBody(request, ResponseBody::bytes);
    }

    private Object getResponseBody(Request request, Command command) {
        try {
            log.debug(URLDecoder.decode(request.toString(), StandardCharsets.UTF_8.name()));
        } catch (UnsupportedEncodingException e) {
            throw new ArvadosApiException(e);
        }

        try (Response response = client.newCall(request).execute()) {
            ResponseBody responseBody = response.body();

            if (!response.isSuccessful()) {
                String errorBody = Objects.requireNonNull(responseBody).string();
                if (errorBody == null || errorBody.length() == 0) {
                    throw new ArvadosApiException(String.format("Error code %s with message: %s", response.code(), response.message()));
                }
                ApiError apiError = MAPPER.readValue(errorBody, ApiError.class);
                throw new ArvadosApiException(String.format("Error code %s with messages: %s", response.code(), apiError.getErrors()));
            }
            return command.readResponseBody(responseBody);
        } catch (IOException e) {
            throw new ArvadosApiException(e);
        }
    }

    private interface Command {
        Object readResponseBody(ResponseBody body) throws IOException;
    }
}
