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

public class ExternalConfigProviderTest {

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
    public void testAutoFetchWebDAVConfiguration() {
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
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then
        assertThat(provider.getKeepWebHost()).isEqualTo("download.example.com");
        assertThat(provider.getKeepWebPort()).isEqualTo(9000);
    }

    @Test
    public void testAutoFetchWebDAVConfigurationWithDefaultHttpsPort() {
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
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then
        assertThat(provider.getKeepWebHost()).isEqualTo("download.example.com");
        assertThat(provider.getKeepWebPort()).isEqualTo(443);
    }

    @Test
    public void testAutoFetchWebDAVConfigurationWithDefaultHttpPort() {
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
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then
        assertThat(provider.getKeepWebHost()).isEqualTo("download.example.com");
        assertThat(provider.getKeepWebPort()).isEqualTo(80);
    }

    @Test
    public void testManualConfigurationTakesPrecedence() {
        // Given - server returns config but we provide manual values
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "    \"WebDAVDownload\": {\n" +
                                "      \"ExternalURL\": \"https://auto.example.com/\"\n" +
                                "    }\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When - manual configuration is provided
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .keepWebHost("manual.example.com")
                .keepWebPort(8080)
                .build();

        // Then - manual values should be used
        assertThat(provider.getKeepWebHost()).isEqualTo("manual.example.com");
        assertThat(provider.getKeepWebPort()).isEqualTo(8080);
    }

    @Test
    public void testAutoFetchDisabled() {
        // When - auto-fetch is explicitly disabled
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost("api.example.com")
                .apiPort(443)
                .apiProtocol("https")
                .apiToken("test-token")
                .autoFetchWebDAV(false)
                .build();

        // Then - keepWeb values should be null/0
        assertThat(provider.getKeepWebHost()).isNull();
        assertThat(provider.getKeepWebPort()).isEqualTo(0);
    }

    @Test
    public void testHandlesApiError() {
        // Given - server returns error
        mockServer.enqueue(new MockResponse()
                .setResponseCode(500)
                .setBody("Internal Server Error"));

        // When
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then - should handle gracefully, keepWeb values should be null/0
        assertThat(provider.getKeepWebHost()).isNull();
        assertThat(provider.getKeepWebPort()).isEqualTo(0);
    }

    @Test
    public void testHandlesMalformedResponse() {
        // Given - server returns malformed JSON
        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody("{ invalid json ]")
                .addHeader("Content-Type", "application/json"));

        // When
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then - should handle gracefully
        assertThat(provider.getKeepWebHost()).isNull();
        assertThat(provider.getKeepWebPort()).isEqualTo(0);
    }

    @Test
    public void testHandlesMissingWebDAVInResponse() {
        // Given - server returns config without WebDAV section
        String configResponse = "{\n" +
                                "  \"Services\": {\n" +
                                "  }\n" +
                                "}";

        mockServer.enqueue(new MockResponse()
                .setResponseCode(200)
                .setBody(configResponse)
                .addHeader("Content-Type", "application/json"));

        // When
        ExternalConfigProvider provider = ExternalConfigProvider.builder()
                .apiHost(mockServer.getHostName())
                .apiPort(mockServer.getPort())
                .apiProtocol("http")
                .apiToken("test-token")
                .build();

        // Then - should handle gracefully
        assertThat(provider.getKeepWebHost()).isNull();
        assertThat(provider.getKeepWebPort()).isEqualTo(0);
    }
}