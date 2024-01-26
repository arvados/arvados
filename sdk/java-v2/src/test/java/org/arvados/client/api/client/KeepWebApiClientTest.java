/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;

import okhttp3.mockwebserver.MockResponse;
import okio.Buffer;

import static org.arvados.client.test.utils.ApiClientTestUtils.getResponse;
import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertNotNull;

public class KeepWebApiClientTest extends ArvadosClientMockedWebServerTest {

    private final KeepWebApiClient client = new KeepWebApiClient(CONFIG);

    @Test
    public void uploadFile() throws Exception {
        // given
        String collectionUuid = "112ci-4zz18-p51w7z3fpopo6sm";
        File file = Files.createTempFile("keep-upload-test", "txt").toFile();
        Files.write(file.toPath(), "test data".getBytes());

        server.enqueue(getResponse("keep-client-upload-response"));

        // when
        String uploadResponse = client.upload(collectionUuid, file, uploadedBytes -> System.out.printf("Uploaded bytes: %s/%s%n", uploadedBytes, file.length()));

        // then
        assertThat(uploadResponse).isEqualTo("Created");
    }

    @Test
    public void downloadPartialIsPerformedSuccessfully() throws Exception {
        // given
        String collectionUuid = "some-collection-uuid";
        String filePathName = "sample-file-path";
        long start = 1024;
        Long end = null;

        byte[] expectedData = "test data".getBytes();

        try (Buffer buffer = new Buffer().write(expectedData)) {
            server.enqueue(new MockResponse().setBody(buffer));

            // when
            InputStream inputStream = client.get(collectionUuid, filePathName, start, end);
            byte[] actualData = inputStreamToByteArray(inputStream);

            // then
            assertNotNull(actualData);
            assertArrayEquals(expectedData, actualData);
        }
    }

    private byte[] inputStreamToByteArray(InputStream inputStream) throws IOException {
        ByteArrayOutputStream buffer = new ByteArrayOutputStream();
        int nRead;
        byte[] data = new byte[1024];
        while ((nRead = inputStream.read(data, 0, data.length)) != -1) {
            buffer.write(data, 0, nRead);
        }
        buffer.flush();
        return buffer.toByteArray();
    }

}
