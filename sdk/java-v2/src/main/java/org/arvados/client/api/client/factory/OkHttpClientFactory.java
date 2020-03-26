/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client.factory;

import com.google.common.base.Suppliers;
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
import java.util.function.Supplier;

/**
 * {@link OkHttpClient} instance factory that builds and configures client instances sharing
 * the common resource pool: this is the recommended approach to optimize resource usage.
 */
public final class OkHttpClientFactory {
    public static final OkHttpClientFactory INSTANCE = new OkHttpClientFactory();
    private final Logger log = org.slf4j.LoggerFactory.getLogger(OkHttpClientFactory.class);
    private final OkHttpClient clientSecure = new OkHttpClient();
    private final Supplier<OkHttpClient> clientUnsecure =
            Suppliers.memoize(this::getDefaultClientAcceptingAllCertificates);

    private OkHttpClientFactory() { /* singleton */}

    public OkHttpClient create(boolean apiHostInsecure) {
        return apiHostInsecure ? getDefaultUnsecureClient() : getDefaultClient();
    }

    /**
     * @return default secure {@link OkHttpClient} with shared resource pool.
     */
    public OkHttpClient getDefaultClient() {
        return clientSecure;
    }

    /**
     * @return default {@link OkHttpClient} with shared resource pool
     * that will accept all SSL certificates by default.
     */
    public OkHttpClient getDefaultUnsecureClient() {
        return clientUnsecure.get();
    }

    /**
     * @return default {@link OkHttpClient.Builder} with shared resource pool.
     */
    public OkHttpClient.Builder getDefaultClientBuilder() {
        return clientSecure.newBuilder();
    }

    /**
     * @return default {@link OkHttpClient.Builder} with shared resource pool
     * that is preconfigured to accept all SSL certificates.
     */
    public OkHttpClient.Builder getDefaultUnsecureClientBuilder() {
        return clientUnsecure.get().newBuilder();
    }

    private OkHttpClient getDefaultClientAcceptingAllCertificates() {
        log.warn("Creating unsafe OkHttpClient. All SSL certificates will be accepted.");
        try {
            // Create a trust manager that does not validate certificate chains
            final TrustManager[] trustAllCerts = {createX509TrustManager()};

            // Install the all-trusting trust manager
            SSLContext sslContext = SSLContext.getInstance("SSL");
            sslContext.init(null, trustAllCerts, new SecureRandom());
            // Create an ssl socket factory with our all-trusting manager
            final SSLSocketFactory sslSocketFactory = sslContext.getSocketFactory();

            // Create the OkHttpClient.Builder with shared resource pool
            final OkHttpClient.Builder builder = clientSecure.newBuilder();
            builder.sslSocketFactory(sslSocketFactory, (X509TrustManager) trustAllCerts[0]);
            builder.hostnameVerifier((hostname, session) -> true);
            return builder.build();
        } catch (NoSuchAlgorithmException | KeyManagementException e) {
            throw new ArvadosClientException("Error establishing SSL context", e);
        }
    }

    private static X509TrustManager createX509TrustManager() {
        return new X509TrustManager() {

            @Override
            public void checkClientTrusted(X509Certificate[] chain, String authType) {
            }

            @Override
            public void checkServerTrusted(X509Certificate[] chain, String authType) {
            }

            @Override
            public X509Certificate[] getAcceptedIssuers() {
                return new X509Certificate[]{};
            }
        };
    }
}
