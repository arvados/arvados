/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import okhttp3.mockwebserver.MockResponse;
import okio.Buffer;
import org.apache.commons.io.FileUtils;
import org.arvados.client.config.FileConfigProvider;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.exception.ArvadosClientException;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.junit.Assert;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.MockitoJUnitRunner;

import java.io.File;

import static junit.framework.TestCase.fail;
import static org.arvados.client.test.utils.ApiClientTestUtils.getResponse;
import static org.assertj.core.api.Assertions.assertThat;

@RunWith(MockitoJUnitRunner.class)
public class KeepClientTest extends ArvadosClientMockedWebServerTest {

    private ConfigProvider configProvider = new FileConfigProvider();
    private static final String TEST_FILE_PATH ="src/test/resources/org/arvados/client/api/client/keep-client-test-file.txt";

    @InjectMocks
    private KeepClient keepClient  = new KeepClient(configProvider);

    @Mock
    private KeepLocator keepLocator;

    @Test
    public void uploadedFile() throws Exception {
        // given
        server.enqueue(getResponse("keep-services-accessible"));
        server.enqueue(new MockResponse().setBody("0887c78c7d6c1a60ac0b3709a4302ee4"));

        // when
        String actual = keepClient.put(new File(TEST_FILE_PATH), 1, 0);

        // then
        assertThat(actual).isEqualTo("0887c78c7d6c1a60ac0b3709a4302ee4");
    }

    @Test
    public void fileIsDownloaded() throws Exception {
        //given
        File data = new File(TEST_FILE_PATH);
        byte[] fileBytes = FileUtils.readFileToByteArray(data);

        // when
        server.enqueue(getResponse("keep-services-accessible"));
        server.enqueue(new MockResponse().setBody(new Buffer().write(fileBytes)));

        byte[] actual = keepClient.getDataChunk(keepLocator);

        Assert.assertArrayEquals(fileBytes, actual);
    }

    @Test
    public void fileIsDownloadedWhenFirstServerDoesNotRespond() throws Exception {
        // given
        File data = new File(TEST_FILE_PATH);
        byte[] fileBytes = FileUtils.readFileToByteArray(data);
        server.enqueue(getResponse("keep-services-accessible")); // two servers accessible
        server.enqueue(new MockResponse().setResponseCode(404)); // first one not responding
        server.enqueue(new MockResponse().setBody(new Buffer().write(fileBytes))); // second one responding

        //when
        byte[] actual = keepClient.getDataChunk(keepLocator);

        //then
        Assert.assertArrayEquals(fileBytes, actual);
    }

    @Test
    public void exceptionIsThrownWhenNoServerResponds() throws Exception {
        //given
        File data = new File(TEST_FILE_PATH);
        server.enqueue(getResponse("keep-services-accessible")); // two servers accessible
        server.enqueue(new MockResponse().setResponseCode(404)); // first one not responding
        server.enqueue(new MockResponse().setResponseCode(404)); // second one not responding

        try {
            //when
            keepClient.getDataChunk(keepLocator);
            fail();
        } catch (ArvadosClientException e) {
            //then
            Assert.assertEquals("No server responding. Unable to download data chunk.", e.getMessage());
        }
    }

    @Test
    public void exceptionIsThrownWhenThereAreNoServersAccessible() throws Exception {
        //given
        server.enqueue(getResponse("keep-services-not-accessible")); // no servers accessible

        try {
            //when
            keepClient.getDataChunk(keepLocator);
            fail();
        } catch (ArvadosClientException e) {
            //then
            Assert.assertEquals("No gateway services available!", e.getMessage());
        }
    }
}
