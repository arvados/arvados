/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import org.arvados.client.api.client.CollectionsApiClient;
import org.arvados.client.api.client.KeepWebApiClient;
import org.arvados.client.api.model.Collection;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.logic.collection.CollectionFactory;
import org.slf4j.Logger;

import java.io.File;
import java.util.List;

public class FileUploader {

    private final KeepWebApiClient keepWebApiClient;
    private final CollectionsApiClient collectionsApiClient;
    private final ConfigProvider config;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(FileUploader.class);

    public FileUploader(KeepWebApiClient keepWebApiClient, CollectionsApiClient collectionsApiClient, ConfigProvider config) {
        this.keepWebApiClient = keepWebApiClient;
        this.collectionsApiClient = collectionsApiClient;
        this.config = config;
    }

    public Collection upload(List<File> sourceFiles, String collectionName, String projectUuid) {
        Collection newCollection = CollectionFactory.builder()
                .config(config)
                .name(collectionName)
                .projectUuid(projectUuid)
                .build()
                .create();

        newCollection = collectionsApiClient.create(newCollection);
        String newCollectionId = newCollection.getUuid();

        sourceFiles.forEach(file -> uploadFile(newCollectionId, file));

        return collectionsApiClient.get(newCollection.getUuid());
    }

    private void uploadFile(String collectionUuid, File file) {
        keepWebApiClient.upload(collectionUuid, file, (progress) -> log.info("Uploaded {} bytes for file: {}", progress, file.getName()));
    }

    public Collection uploadToExistingCollection(List<File> files, String collectionUuid) {
        files.forEach(file -> uploadFile(collectionUuid, file));

        return collectionsApiClient.get(collectionUuid);
    }

}
