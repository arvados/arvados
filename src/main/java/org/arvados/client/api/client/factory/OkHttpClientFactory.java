/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client.factory;

import okhttp3.OkHttpClient;
import org.arvados.client.exception.ArvadosClientException;
import org.slf4j.Logger;

import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import java.security.KeyManagementException;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.security.cert.X509Certificate;

public class OkHttpClientFactory {

    private final Logger log = org.slf4j.LoggerFactory.getLogger(OkHttpClientFactory.class);

    OkHttpClientFactory() {
    }

    public static OkHttpClientFactoryBuilder builder() {
        return new OkHttpClientFactoryBuilder();
    }

    public OkHttpClient create(boolean apiHostInsecure) {
        OkHttpClient.Builder builder = new OkHttpClient.Builder();
        if (apiHostInsecure) {
            trustAllCertificates(builder);
        }
        return builder.build();
    }

    private void trustAllCertificates(OkHttpClient.Builder builder) {
        log.warn("Creating unsafe OkHttpClient. All SSL certificates will be accepted.");
        try {
            // Create a trust manager that does not validate certificate chains
            final TrustManager[] trustAllCerts = new TrustManager[] { createX509TrustManager() };

            // Install the all-trusting trust manager
            SSLContext sslContext = SSLContext.getInstance("SSL");
            sslContext.init(null, trustAllCerts, new SecureRandom());
            // Create an ssl socket factory with our all-trusting manager
            final SSLSocketFactory sslSocketFactory = sslContext.getSocketFactory();

            builder.sslSocketFactory(sslSocketFactory, (X509TrustManager) trustAllCerts[0]);
            builder.hostnameVerifier((hostname, session) -> true);
        } catch (NoSuchAlgorithmException | KeyManagementException e) {
            throw new ArvadosClientException("Error establishing SSL context", e);
        }
    }

    private static X509TrustManager createX509TrustManager() {
        return new X509TrustManager() {
            
            @Override
            public void checkClientTrusted(X509Certificate[] chain, String authType) {}

            @Override
            public void checkServerTrusted(X509Certificate[] chain, String authType) {}

            @Override
            public X509Certificate[] getAcceptedIssuers() {
                return new X509Certificate[] {};
            }
        };
    }

    public static class OkHttpClientFactoryBuilder {
        OkHttpClientFactoryBuilder() {
        }

        public OkHttpClientFactory build() {
            return new OkHttpClientFactory();
        }

        public String toString() {
            return "OkHttpClientFactory.OkHttpClientFactoryBuilder()";
        }
    }
}
