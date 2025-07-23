/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client.factory;

import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import okhttp3.mockwebserver.MockResponse;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Assert;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.junit.MockitoJUnitRunner;

import javax.net.ssl.KeyManagerFactory;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;
import javax.net.ssl.TrustManagerFactory;
import java.io.FileInputStream;
import java.security.KeyStore;


@RunWith(MockitoJUnitRunner.class)
public class OkHttpClientFactoryTest extends ArvadosClientMockedWebServerTest {

    @Test(expected = javax.net.ssl.SSLHandshakeException.class)
    public void secureOkHttpClientIsCreated() throws Exception {

        // given
        OkHttpClientFactory factory = OkHttpClientFactory.INSTANCE;
        // * configure HTTPS server
        SSLSocketFactory sf = getSSLSocketFactoryWithSelfSignedCertificate();
        server.useHttps(sf, false);
        server.enqueue(new MockResponse().setBody("OK"));
        // * prepare client HTTP request
        Request request = new Request.Builder()
                .url("https://localhost:9000/")
                .build();

        // when - then (SSL certificate is verified)
        OkHttpClient actual = factory.create(false);
        Response response = actual.newCall(request).execute();
    }

    @Test
    public void insecureOkHttpClientIsCreated() throws Exception {
        // given
        OkHttpClientFactory factory = OkHttpClientFactory.INSTANCE;
        // * configure HTTPS server
        SSLSocketFactory sf = getSSLSocketFactoryWithSelfSignedCertificate();
        server.useHttps(sf, false);
        server.enqueue(new MockResponse().setBody("OK"));
        // * prepare client HTTP request
        Request request = new Request.Builder()
                .url("https://localhost:9000/")
                .build();

        // when (SSL certificate is not verified)
        OkHttpClient actual = factory.create(true);
        Response response = actual.newCall(request).execute();

        // then
        Assert.assertEquals(response.body().string(),"OK");
    }


    /*
        This ugly boilerplate is needed to enable self signed certificate.

        It requires selfsigned.keystore.jks file. It was generated with:
        keytool -genkey -v -keystore mystore.keystore.jks -alias alias_name -keyalg RSA -keysize 2048 -validity 10000
     */
    public SSLSocketFactory getSSLSocketFactoryWithSelfSignedCertificate() throws Exception {

        FileInputStream stream = new FileInputStream("src/test/resources/selfsigned.keystore.jks");
        char[] serverKeyStorePassword = "123456".toCharArray();
        KeyStore serverKeyStore = KeyStore.getInstance(KeyStore.getDefaultType());
        serverKeyStore.load(stream, serverKeyStorePassword);

        String kmfAlgorithm = KeyManagerFactory.getDefaultAlgorithm();
        KeyManagerFactory kmf = KeyManagerFactory.getInstance(kmfAlgorithm);
        kmf.init(serverKeyStore, serverKeyStorePassword);

        TrustManagerFactory trustManagerFactory = TrustManagerFactory.getInstance(kmfAlgorithm);
        trustManagerFactory.init(serverKeyStore);

        SSLContext sslContext = SSLContext.getInstance("SSL");
        sslContext.init(kmf.getKeyManagers(), trustManagerFactory.getTrustManagers(), null);
        return sslContext.getSocketFactory();
    }
}
