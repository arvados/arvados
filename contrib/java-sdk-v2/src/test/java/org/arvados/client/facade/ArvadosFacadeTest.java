/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.facade;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.ObjectWriter;
import okhttp3.mockwebserver.MockResponse;
import okio.Buffer;
import org.apache.commons.io.FileUtils;
import org.arvados.client.api.model.Collection;
import org.arvados.client.api.model.KeepService;
import org.arvados.client.api.model.KeepServiceList;
import org.arvados.client.common.Characters;
import org.arvados.client.test.utils.ArvadosClientMockedWebServerTest;
import org.arvados.client.test.utils.FileTestUtils;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.Ignore;

import java.io.File;
import java.nio.charset.Charset;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

import static org.arvados.client.test.utils.ApiClientTestUtils.getResponse;
import static org.arvados.client.test.utils.FileTestUtils.*;
import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

public class ArvadosFacadeTest extends ArvadosClientMockedWebServerTest {

    ArvadosFacade facade = new ArvadosFacade(CONFIG);

    @Before
    public void setUp() throws Exception {
        FileTestUtils.createDirectory(FILE_SPLIT_TEST_DIR);
        FileTestUtils.createDirectory(FILE_DOWNLOAD_TEST_DIR);
    }

    @Test
    @Ignore("Failing test #15041")
    public void uploadIsPerformedSuccessfullyUsingDiskOnlyKeepServices() throws Exception {

        // given
        String keepServicesAccessible = setMockedServerPortToKeepServices("keep-services-accessible-disk-only");
        server.enqueue(new MockResponse().setBody(keepServicesAccessible));

        String blockLocator = "7df44272090cee6c0732382bba415ee9";
        String signedBlockLocator = blockLocator + "+70+A189a93acda6e1fba18a9dffd42b6591cbd36d55d@5a1c17b6";
        for (int i = 0; i < 8; i++) {
            server.enqueue(new MockResponse().setBody(signedBlockLocator));
        }
        server.enqueue(getResponse("users-get"));
        server.enqueue(getResponse("collections-create-manifest"));

        FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_FOURTH_GB);

        // when
        Collection actual = facade.upload(Arrays.asList(new File(TEST_FILE)), "Super Collection", null);

        // then
        assertThat(actual.getName()).contains("Super Collection");
    }

    @Test
    public void uploadIsPerformedSuccessfully() throws Exception {

        // given
        // First response: get current user (called by CollectionFactory when projectUuid is null)
        server.enqueue(getResponse("users-get"));

        // Second response: create collection
        server.enqueue(getResponse("collections-create-manifest"));

        // Third response: upload file to KeepWeb (it returns empty response)
        server.enqueue(new MockResponse().setBody(""));

        // Fourth response: get the updated collection
        server.enqueue(getResponse("collections-create-manifest"));

        FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_FOURTH_GB);

        // when
        Collection actual = facade.upload(Arrays.asList(new File(TEST_FILE)), "Super Collection", null);

        // then
        assertThat(actual.getName()).contains("Super Collection");
    }

    @Test
    public void downloadOfWholeCollectionIsPerformedSuccessfully() throws Exception {

        //given
        String collectionUuid = "ardev-4zz18-jk5vo4uo9u5vj52";
        server.enqueue(getResponse("collections-download-file"));

        // Mock KeepWeb API responses for each file
        List<File> files = generatePredefinedFiles();
        for (File f : files) {
            server.enqueue(new MockResponse().setBody(new Buffer().write(Files.readAllBytes(f.toPath()))));
        }

        //when
        List<File> downloadedFiles = facade.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, false);

        //then
        File collectionDestination = new File(FILE_DOWNLOAD_TEST_DIR + Characters.SLASH + collectionUuid);
        assertEquals(3, downloadedFiles.size());
        assertTrue(collectionDestination.exists());
        assertThat(downloadedFiles).allMatch(File::exists);
        assertEquals(files.stream().map(File::getName).collect(Collectors.toList()), downloadedFiles.stream().map(File::getName).collect(Collectors.toList()));
        assertEquals(files.stream().map(File::length).collect(Collectors.toList()), downloadedFiles.stream().map(File::length).collect(Collectors.toList()));
    }

    @Test
    public void downloadOfWholeCollectionUsingKeepWebPerformedSuccessfully() throws Exception {

        //given
        String collectionUuid = "ardev-4zz18-jk5vo4uo9u5vj52";
        server.enqueue(getResponse("collections-download-file"));

        List<File> files = generatePredefinedFiles();
        for (File f : files) {
            server.enqueue(new MockResponse().setBody(new Buffer().write(FileUtils.readFileToByteArray(f))));
        }

        //when
        List<File> downloadedFiles = facade.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, true);

        //then
        assertEquals(3, downloadedFiles.size());
        assertThat(downloadedFiles).allMatch(File::exists);
        assertEquals(files.stream().map(File::getName).collect(Collectors.toList()), downloadedFiles.stream().map(File::getName).collect(Collectors.toList()));
        assertTrue(downloadedFiles.stream().map(File::length).collect(Collectors.toList()).containsAll(files.stream().map(File::length).collect(Collectors.toList())));
    }

    @Test
    public void downloadOfSingleFilePerformedSuccessfully() throws Exception {

        //given
        String collectionUuid = "ardev-4zz18-jk5vo4uo9u5vj52";
        server.enqueue(getResponse("collections-download-file"));

        File file = generatePredefinedFiles().get(0);
        byte[] fileData = FileUtils.readFileToByteArray(file);
        server.enqueue(new MockResponse().setBody(new Buffer().write(fileData)));

        //when
        File downloadedFile = facade.downloadFile(file.getName(), collectionUuid, FILE_DOWNLOAD_TEST_DIR);

        //then
        assertTrue(downloadedFile.exists());
        assertEquals(file.getName(), downloadedFile.getName());
        assertEquals(file.length(), downloadedFile.length());
    }

    private String setMockedServerPortToKeepServices(String jsonPath) throws Exception {

        ObjectMapper mapper = new ObjectMapper().findAndRegisterModules();
        String filePath = String.format("src/test/resources/org/arvados/client/api/client/%s.json", jsonPath);
        File jsonFile = new File(filePath);
        String json = FileUtils.readFileToString(jsonFile, Charset.defaultCharset());
        KeepServiceList keepServiceList = mapper.readValue(json, KeepServiceList.class);
        List<KeepService> items = keepServiceList.getItems();
        for (KeepService keepService : items) {
            keepService.setServicePort(server.getPort());
        }
        ObjectWriter writer = mapper.writer().withDefaultPrettyPrinter();
        return writer.writeValueAsString(keepServiceList);
    }

    //Method to copy multiple byte[] arrays into one byte[] array
    private byte[] addAll(byte[] array1, byte[] array2) {
        byte[] joinedArray = new byte[array1.length + array2.length];
        System.arraycopy(array1, 0, joinedArray, 0, array1.length);
        System.arraycopy(array2, 0, joinedArray, array1.length, array2.length);
        return joinedArray;
    }

    @After
    public void tearDown() throws Exception {
        FileTestUtils.cleanDirectory(FILE_SPLIT_TEST_DIR);
        FileTestUtils.cleanDirectory(FILE_DOWNLOAD_TEST_DIR);
    }
}
