/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import org.arvados.client.api.client.factory.OkHttpClientFactory;
import org.arvados.client.api.model.ArvadosConfig;
import org.arvados.client.exception.ArvadosApiException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.concurrent.TimeUnit;

public class ConfigApiClient {

    private static final Logger log = LoggerFactory.getLogger(ConfigApiClient.class);
    private static final ObjectMapper MAPPER = new ObjectMapper().findAndRegisterModules();
    private static final String CONFIG_ENDPOINT = "/arvados/v1/config";

    private final OkHttpClient client;
    private final String baseUrl;

    public ConfigApiClient(String protocol, String host, int port, boolean insecure) {
        this.baseUrl = String.format("%s://%s:%d", protocol, host, port);
        this.client = OkHttpClientFactory.INSTANCE.create(insecure)
                .newBuilder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(10, TimeUnit.SECONDS)
                .build();
    }

    public ArvadosConfig fetchConfig() throws ArvadosApiException {
        String url = baseUrl + CONFIG_ENDPOINT;
        Request request = new Request.Builder()
                .url(url)
                .get()
                .build();

        try (Response response = client.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                String errorMessage = String.format("Failed to fetch config from %s. Status: %d",
                        url, response.code());
                log.error(errorMessage);
                throw new ArvadosApiException(errorMessage);
            }

            String responseBody = response.body() != null ? response.body().string() : "";
            return MAPPER.readValue(responseBody, ArvadosConfig.class);

        } catch (IOException e) {
            String errorMessage = String.format("Error fetching config from %s: %s",
                    url, e.getMessage());
            log.error(errorMessage, e);
            throw new ArvadosApiException(errorMessage, e);
        }
    }
}