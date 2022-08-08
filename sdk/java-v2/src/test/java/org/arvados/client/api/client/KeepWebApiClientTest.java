/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.client;

import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Test;

import java.io.File;
import java.nio.file.Files;

import static org.arvados.client.test.utils.ApiClientTestUtils.getResponse;
import static org.assertj.core.api.Assertions.assertThat;

public class KeepWebApiClientTest extends ArvadosClientMockedWebServerTest {

    private KeepWebApiClient client = new KeepWebApiClient(CONFIG);

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

}
