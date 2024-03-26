/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.facade;

import org.apache.commons.io.FileUtils;
import org.arvados.client.api.model.Collection;
import org.arvados.client.common.Characters;
import org.arvados.client.config.ExternalConfigProvider;
import org.arvados.client.junit.categories.IntegrationTests;
import org.arvados.client.logic.collection.FileToken;
import org.arvados.client.test.utils.ArvadosClientIntegrationTest;
import org.arvados.client.test.utils.FileTestUtils;
import org.junit.After;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;
import org.junit.experimental.categories.Category;

import java.io.File;
import java.util.Collections;
import java.util.List;
import java.util.UUID;

import static org.arvados.client.test.utils.FileTestUtils.FILE_DOWNLOAD_TEST_DIR;
import static org.arvados.client.test.utils.FileTestUtils.FILE_SPLIT_TEST_DIR;
import static org.arvados.client.test.utils.FileTestUtils.TEST_FILE;
import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

@Category(IntegrationTests.class)
public class ArvadosFacadeIntegrationTest extends ArvadosClientIntegrationTest {


    private static final String COLLECTION_NAME = "Test collection " + UUID.randomUUID().toString();
    private String collectionUuid;

    @Before
    public void setUp() throws Exception {
        FileTestUtils.createDirectory(FILE_SPLIT_TEST_DIR);
        FileTestUtils.createDirectory(FILE_DOWNLOAD_TEST_DIR);
    }

    @Test
    public void uploadOfFileIsPerformedSuccessfully() throws Exception {
        // given
        File file = FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_FOURTH_GB / 200);

        // when
        Collection actual = FACADE.upload(Collections.singletonList(file), COLLECTION_NAME, PROJECT_UUID);
        collectionUuid = actual.getUuid();

        // then
        assertThat(actual.getName()).contains("Test collection");
        assertThat(actual.getManifestText()).contains(file.length() + Characters.COLON + file.getName());
    }

    @Test
    public void uploadOfFilesIsPerformedSuccessfully() throws Exception {
        // given
        List<File> files = FileTestUtils.generatePredefinedFiles();
        files.addAll(FileTestUtils.generatePredefinedFiles());

        // when
        Collection actual = FACADE.upload(files, COLLECTION_NAME, PROJECT_UUID);
        collectionUuid = actual.getUuid();

        // then
        assertThat(actual.getName()).contains("Test collection");
        files.forEach(f -> assertThat(actual.getManifestText()).contains(f.length() + Characters.COLON + f.getName().replace(" ", Characters.SPACE)));
    }

    @Test
    public void uploadToExistingCollectionIsPerformedSuccessfully() throws Exception {
        // given
        File file = FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_EIGTH_GB / 500);
        Collection existing = createTestCollection();

        // when
        Collection actual = FACADE.uploadToExistingCollection(Collections.singletonList(file), collectionUuid);

        // then
        assertEquals(collectionUuid, actual.getUuid());
        assertThat(actual.getManifestText()).contains(file.length() + Characters.COLON + file.getName());
    }

    @Test
    public void uploadWithExternalConfigProviderWorksProperly() throws Exception {
        //given
        ArvadosFacade facade = new ArvadosFacade(buildExternalConfig());
        File file = FileTestUtils.generateFile(TEST_FILE, FileTestUtils.ONE_FOURTH_GB / 200);

        //when
        Collection actual = facade.upload(Collections.singletonList(file), COLLECTION_NAME, PROJECT_UUID);
        collectionUuid = actual.getUuid();

        //then
        assertThat(actual.getName()).contains("Test collection");
        assertThat(actual.getManifestText()).contains(file.length() + Characters.COLON + file.getName());
    }

    @Test
    public void creationOfEmptyCollectionPerformedSuccesfully() {
        // given
        String collectionName = "Empty collection " + UUID.randomUUID().toString();

        // when
        Collection actual = FACADE.createEmptyCollection(collectionName, PROJECT_UUID);
        collectionUuid = actual.getUuid();

        // then
        assertEquals(collectionName, actual.getName());
        assertEquals(PROJECT_UUID, actual.getOwnerUuid());
    }

    @Test
    public void fileTokensAreListedFromCollection() throws Exception {
        //given
        List<File> files = uploadTestFiles();

        //when
        List<FileToken> actual = FACADE.listFileInfoFromCollection(collectionUuid);

        //then
        assertEquals(files.size(), actual.size());
        for (int i = 0; i < files.size(); i++) {
            assertEquals(files.get(i).length(), actual.get(i).getFileSize());
        }
    }

    @Test
    public void downloadOfFilesPerformedSuccessfully() throws Exception {
        //given
        List<File> files = uploadTestFiles();
        File destination = new File(FILE_DOWNLOAD_TEST_DIR + Characters.SLASH + collectionUuid);

        //when
        List<File> actual = FACADE.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, false);

        //then
        assertEquals(files.size(), actual.size());
        assertTrue(destination.exists());
        assertThat(actual).allMatch(File::exists);
        for (int i = 0; i < files.size(); i++) {
            assertEquals(files.get(i).length(), actual.get(i).length());
        }
    }

    @Test
    public void downloadOfFilesPerformedSuccessfullyUsingKeepWeb() throws Exception {
        //given
        List<File> files = uploadTestFiles();
        File destination = new File(FILE_DOWNLOAD_TEST_DIR + Characters.SLASH + collectionUuid);

        //when
        List<File> actual = FACADE.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, true);

        //then
        assertEquals(files.size(), actual.size());
        assertTrue(destination.exists());
        assertThat(actual).allMatch(File::exists);
        for (int i = 0; i < files.size(); i++) {
            assertEquals(files.get(i).length(), actual.get(i).length());
        }
    }

    @Test
    public void singleFileIsDownloadedSuccessfullyUsingKeepWeb() throws Exception {
        //given
        File file = uploadSingleTestFile(false);

        //when
        File actual = FACADE.downloadFile(file.getName(), collectionUuid, FILE_DOWNLOAD_TEST_DIR);

        //then
        assertThat(actual).exists();
        assertThat(actual.length()).isEqualTo(file.length());
    }

    @Test
    public void downloadOfOneFileSplittedToMultipleLocatorsPerformedSuccesfully() throws Exception {
        //given
        File file = uploadSingleTestFile(true);

        List<File> actual = FACADE.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, false);

        Assert.assertEquals(1, actual.size());
        assertThat(actual.get(0).length()).isEqualTo(file.length());
    }

    @Test
    public void downloadWithExternalConfigProviderWorksProperly() throws Exception {
        //given
        ArvadosFacade facade = new ArvadosFacade(buildExternalConfig());
        List<File> files = uploadTestFiles();
        //when
        List<File> actual = facade.downloadCollectionFiles(collectionUuid, FILE_DOWNLOAD_TEST_DIR, false);

        //then
        assertEquals(files.size(), actual.size());
        assertThat(actual).allMatch(File::exists);
        for (int i = 0; i < files.size(); i++) {
            assertEquals(files.get(i).length(), actual.get(i).length());
        }
    }

    private ExternalConfigProvider buildExternalConfig() {
        return ExternalConfigProvider
                .builder()
                .apiHostInsecure(CONFIG.isApiHostInsecure())
                .keepWebHost(CONFIG.getKeepWebHost())
                .keepWebPort(CONFIG.getKeepWebPort())
                .apiHost(CONFIG.getApiHost())
                .apiPort(CONFIG.getApiPort())
                .apiToken(CONFIG.getApiToken())
                .apiProtocol(CONFIG.getApiProtocol())
                .fileSplitSize(CONFIG.getFileSplitSize())
                .fileSplitDirectory(CONFIG.getFileSplitDirectory())
                .numberOfCopies(CONFIG.getNumberOfCopies())
                .numberOfRetries(CONFIG.getNumberOfRetries())
                .connectTimeout(CONFIG.getConnectTimeout())
                .readTimeout(CONFIG.getReadTimeout())
                .writeTimeout(CONFIG.getWriteTimeout())
                .build();
    }

    private Collection createTestCollection() {
        Collection collection = FACADE.createEmptyCollection(COLLECTION_NAME, PROJECT_UUID);
        collectionUuid = collection.getUuid();
        return collection;
    }

    private List<File> uploadTestFiles() throws Exception{
        createTestCollection();
        List<File> files = FileTestUtils.generatePredefinedFiles();
        FACADE.uploadToExistingCollection(files, collectionUuid);
        return files;
    }

    private File uploadSingleTestFile(boolean bigFile) throws Exception{
        createTestCollection();
        Long fileSize = bigFile ? FileUtils.ONE_MB * 70 : FileTestUtils.ONE_EIGTH_GB / 100;
        File file = FileTestUtils.generateFile(TEST_FILE, fileSize);
        FACADE.uploadToExistingCollection(Collections.singletonList(file), collectionUuid);
        return file;
    }

    @After
    public void tearDown() throws Exception {
        FileTestUtils.cleanDirectory(FILE_SPLIT_TEST_DIR);
        FileTestUtils.cleanDirectory(FILE_DOWNLOAD_TEST_DIR);

        if(collectionUuid != null)
        FACADE.deleteCollection(collectionUuid);
    }
}
