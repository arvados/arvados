/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.config;

import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.io.IOException;

import static org.assertj.core.api.Assertions.assertThat;

public class WebDAVConfigFetcherTest {

    private MockWebServer mockServer;

    @Before
    public void setUp() throws IOException {
        mockServer = new MockWebServer();
        mockServer.start();
    }

    @After
    public void tearDown() throws IOException {
        mockServer.shutdown();
    }

    @Test
    public void testFetchWithValidConfig() {
        // Given
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "    \"WebDAVDownload\": {\n" +
                                "      \"ExternalURL\": \"https://download.example.com:9000/\"\n" +
                                "    }\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNotNull();
        assertThat(config.getHost()).isEqualTo("download.example.com");
        assertThat(config.getPort()).isEqualTo(9000);
    }

    @Test
    public void testFetchWithDefaultHttpsPort() {
        // Given
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "    \"WebDAVDownload\": {\n" +
                                "      \"ExternalURL\": \"https://download.example.com/\"\n" +
                                "    }\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNotNull();
        assertThat(config.getHost()).isEqualTo("download.example.com");
        assertThat(config.getPort()).isEqualTo(443);
    }

    @Test
    public void testFetchWithDefaultHttpPort() {
        // Given
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "    \"WebDAVDownload\": {\n" +
                                "      \"ExternalURL\": \"http://download.example.com/\"\n" +
                                "    }\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNotNull();
        assertThat(config.getHost()).isEqualTo("download.example.com");
        assertThat(config.getPort()).isEqualTo(80);
    }

    @Test
    public void testFetchWithApiError() {
        // Given
        mockServer.enqueue(new MockResponse()
                .setResponseCode(500)
                .setBody("Internal Server Error"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }

    @Test
    public void testFetchWithMalformedJson() {
        // Given
        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody("{ invalid json ]")
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }

    @Test
    public void testFetchWithMissingWebDAVSection() {
        // Given
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }

    @Test
    public void testFetchWithNullApiHost() {
        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "https", null, 443, false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }

    @Test
    public void testFetchWithEmptyApiHost() {
        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "https", "", 443, false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }

    @Test
    public void testFetchWithInvalidWebDAVUrl() {
        // Given
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "    \"WebDAVDownload\": {\n" +
                                "      \"ExternalURL\": \"not-a-valid-url\"\n" +
                                "    }\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        WebDAVConfigFetcher fetcher = new WebDAVConfigFetcher(
                "http", mockServer.getHostName(), mockServer.getPort(), false
        );
        WebDAVConfigFetcher.WebDAVConfig config = fetcher.fetch();

        // Then
        assertThat(config).isNull();
    }
}