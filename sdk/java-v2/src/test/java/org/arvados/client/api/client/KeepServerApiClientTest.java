/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import com.google.common.collect.Maps;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.RecordedRequest;
import okio.Buffer;
import org.apache.commons.io.FileUtils;
import org.arvados.client.common.Headers;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import java.io.File;
import java.util.Map;

import static org.arvados.client.test.utils.ApiClientTestUtils.assertAuthorizationHeader;
import static org.assertj.core.api.Assertions.assertThat;

public class KeepServerApiClientTest extends ArvadosClientMockedWebServerTest {

    private KeepServerApiClient client = new KeepServerApiClient(CONFIG);

    @Test
    public void uploadFileToServer() throws Exception {

        // given
        String blockLocator = "7df44272090cee6c0732382bba415ee9";
        String signedBlockLocator = blockLocator + "+70+A189a93acda6e1fba18a9dffd42b6591cbd36d55d@5a1c17b6";
        server.enqueue(new MockResponse().setBody(signedBlockLocator));

        String url = server.url(blockLocator).toString();
        File body = new File("README.md");
        Map<String, String> headers = Maps.newHashMap();
        headers.put(Headers.X_KEEP_DESIRED_REPLICAS, "2");

        // when
        String actual = client.upload(url, headers, body);

        // then
        RecordedRequest request = server.takeRequest();
        assertAuthorizationHeader(request);
        assertThat(request.getPath()).isEqualTo("/" + blockLocator);

        assertThat(actual).isEqualTo(signedBlockLocator);
    }

    @Test
    public void downloadFileFromServer() throws Exception {
        File data = new File("README.md");
        byte[] fileBytes = FileUtils.readFileToByteArray(data);
        server.enqueue(new MockResponse().setBody(new Buffer().write(fileBytes)));

        String blockLocator = "7df44272090cee6c0732382bba415ee9";
        String signedBlockLocator = blockLocator + "+70+A189a93acda6e1fba18a9dffd42b6591cbd36d55d@5a1c17b6";

        String url = server.url(signedBlockLocator).toString();

        byte[] actual = client.download(url);
        RecordedRequest request = server.takeRequest();
        assertThat(request.getPath()).isEqualTo("/" + signedBlockLocator);
        assertThat(actual).isEqualTo(fileBytes);

    }
}
