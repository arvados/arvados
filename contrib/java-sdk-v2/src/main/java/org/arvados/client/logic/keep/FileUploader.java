/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import com.google.common.collect.Lists;
import org.arvados.client.api.client.CollectionsApiClient;
import org.arvados.client.api.model.Collection;
import org.arvados.client.common.Characters;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.exception.ArvadosClientException;
import org.arvados.client.logic.collection.CollectionFactory;
import org.arvados.client.utils.FileMerge;
import org.arvados.client.utils.FileSplit;
import org.slf4j.Logger;

import java.io.File;
import java.io.IOException;
import java.util.List;
import java.util.Objects;
import java.util.UUID;

import static java.util.stream.Collectors.toList;

public class FileUploader {

    private final KeepClient keepClient;
    private final CollectionsApiClient collectionsApiClient;
    private final ConfigProvider config;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(FileUploader.class);

    public FileUploader(KeepClient keepClient, CollectionsApiClient collectionsApiClient, ConfigProvider config) {
        this.keepClient = keepClient;
        this.collectionsApiClient = collectionsApiClient;
        this.config = config;
    }

    public Collection upload(List<File> sourceFiles, String collectionName, String projectUuid) {
        List<String> locators = uploadToKeep(sourceFiles);
        CollectionFactory collectionFactory = CollectionFactory.builder()
                .config(config)
                .name(collectionName)
                .projectUuid(projectUuid)
                .manifestFiles(sourceFiles)
                .manifestLocators(locators)
                .build();

        Collection newCollection = collectionFactory.create();
        return collectionsApiClient.create(newCollection);
    }

    public Collection uploadToExistingCollection(List<File> files, String collectionUuid) {
        List<String> locators = uploadToKeep(files);
        Collection collectionBeforeUpload = collectionsApiClient.get(collectionUuid);
        String oldManifest = collectionBeforeUpload.getManifestText();

        CollectionFactory collectionFactory = CollectionFactory.builder()
                .config(config)
                .manifestFiles(files)
                .manifestLocators(locators).build();

        String newPartOfManifestText = collectionFactory.create().getManifestText();
        String newManifest = oldManifest + newPartOfManifestText;

        collectionBeforeUpload.setManifestText(newManifest);
        return collectionsApiClient.update(collectionBeforeUpload);
    }

    private List<String> uploadToKeep(List<File> files) {
        File targetDir = config.getFileSplitDirectory();
        File combinedFile = new File(targetDir.getAbsolutePath() + Characters.SLASH + UUID.randomUUID());
        List<File> chunks;
        try {
            FileMerge.merge(files, combinedFile);
            chunks = FileSplit.split(combinedFile, targetDir, config.getFileSplitSize());
        } catch (IOException e) {
            throw new ArvadosClientException("Cannot create file chunks for upload", e);
        }
        combinedFile.delete();

        int copies = config.getNumberOfCopies();
        int numRetries = config.getNumberOfRetries();

        List<String> locators = Lists.newArrayList();
        for (File chunk : chunks) {
            try {
                locators.add(keepClient.put(chunk, copies, numRetries));
            } catch (ArvadosClientException e) {
                log.error("Problem occurred while uploading chunk file {}", chunk.getName(), e);
                throw e;
            }
        }
        return locators.stream()
                .filter(Objects::nonNull)
                .collect(toList());
    }
}
