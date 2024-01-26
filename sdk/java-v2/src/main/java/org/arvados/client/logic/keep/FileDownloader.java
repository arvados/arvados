/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import com.google.common.collect.Lists;
import org.arvados.client.api.client.CollectionsApiClient;
import org.arvados.client.api.client.KeepWebApiClient;
import org.arvados.client.api.model.Collection;
import org.arvados.client.common.Characters;
import org.arvados.client.exception.ArvadosClientException;
import org.arvados.client.logic.collection.FileToken;
import org.arvados.client.logic.collection.ManifestDecoder;
import org.arvados.client.logic.collection.ManifestStream;
import org.arvados.client.logic.keep.exception.DownloadFolderAlreadyExistsException;
import org.arvados.client.logic.keep.exception.FileAlreadyExistsException;
import org.slf4j.Logger;

import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.RandomAccessFile;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class FileDownloader {

    private final KeepClient keepClient;
    private final ManifestDecoder manifestDecoder;
    private final CollectionsApiClient collectionsApiClient;
    private final KeepWebApiClient keepWebApiClient;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(FileDownloader.class);

    public FileDownloader(KeepClient keepClient, ManifestDecoder manifestDecoder, CollectionsApiClient collectionsApiClient, KeepWebApiClient keepWebApiClient) {
        this.keepClient = keepClient;
        this.manifestDecoder = manifestDecoder;
        this.collectionsApiClient = collectionsApiClient;
        this.keepWebApiClient = keepWebApiClient;
    }

    public List<FileToken> listFileInfoFromCollection(String collectionUuid) {
        Collection requestedCollection = collectionsApiClient.get(collectionUuid);
        String manifestText = requestedCollection.getManifestText();

        // decode manifest text and get list of all FileTokens for this collection
        return manifestDecoder.decode(manifestText)
                .stream()
                .flatMap(p -> p.getFileTokens().stream())
                .collect(Collectors.toList());
    }

    public File downloadSingleFileUsingKeepWeb(String filePathName, String collectionUuid, String pathToDownloadFolder) {
        FileToken fileToken = getFileTokenFromCollection(filePathName, collectionUuid);
        if (fileToken == null) {
            throw new ArvadosClientException(String.format("%s not found in Collection with UUID %s", filePathName, collectionUuid));
        }

        File downloadedFile = checkIfFileExistsInTargetLocation(fileToken, pathToDownloadFolder);
        try (FileOutputStream fos = new FileOutputStream(downloadedFile)) {
            fos.write(keepWebApiClient.download(collectionUuid, filePathName));
        } catch (IOException e) {
            throw new ArvadosClientException(String.format("Unable to write down file %s", fileToken.getFileName()), e);
        }
        return downloadedFile;
    }

    public File downloadFileWithResume(String collectionUuid, String fileName, String pathToDownloadFolder, long offset, int bufferSize) throws IOException {
        if (bufferSize <= 0) {
            throw new IllegalArgumentException("Buffer size must be greater than 0");
        }

        File destinationFile = new File(pathToDownloadFolder, fileName);

        if (!destinationFile.exists()) {
            boolean isCreated = destinationFile.createNewFile();
            if (!isCreated) {
                throw new IOException("Failed to create new file: " + destinationFile.getAbsolutePath());
            }
        }

        try (RandomAccessFile outputFile = new RandomAccessFile(destinationFile, "rw")) {
            outputFile.seek(offset);

            byte[] buffer = new byte[bufferSize];
            int bytesRead;
            InputStream inputStream = new ByteArrayInputStream(keepWebApiClient.downloadPartial(collectionUuid, fileName, offset));
            while ((bytesRead = inputStream.read(buffer)) != -1) {
                outputFile.write(buffer, 0, bytesRead);
            }
        }

        return destinationFile;
    }

    public List<File> downloadFilesFromCollectionUsingKeepWeb(String collectionUuid, String pathToDownloadFolder) {
        String collectionTargetDir = setTargetDirectory(collectionUuid, pathToDownloadFolder).getAbsolutePath();
        List<FileToken> fileTokens = listFileInfoFromCollection(collectionUuid);

        List<CompletableFuture<File>> futures = Lists.newArrayList();
        for (FileToken fileToken : fileTokens) {
            futures.add(CompletableFuture.supplyAsync(() -> this.downloadOneFileFromCollectionUsingKeepWeb(fileToken, collectionUuid, collectionTargetDir)));
        }

        @SuppressWarnings("unchecked")
        CompletableFuture<File>[] array = futures.toArray(new CompletableFuture[0]);
        return Stream.of(array)
                .map(CompletableFuture::join).collect(Collectors.toList());
    }

    private FileToken getFileTokenFromCollection(String filePathName, String collectionUuid) {
        return listFileInfoFromCollection(collectionUuid)
                .stream()
                .filter(p -> (p.getFullPath()).equals(filePathName))
                .findFirst()
                .orElse(null);
    }

    private File checkIfFileExistsInTargetLocation(FileToken fileToken, String pathToDownloadFolder) {
        String fileName = fileToken.getFileName();

        File downloadFile = new File(pathToDownloadFolder + Characters.SLASH + fileName);
        if (downloadFile.exists()) {
            throw new FileAlreadyExistsException(String.format("File %s exists in location %s", fileName, pathToDownloadFolder));
        } else {
            return downloadFile;
        }
    }

    private File downloadOneFileFromCollectionUsingKeepWeb(FileToken fileToken, String collectionUuid, String pathToDownloadFolder) {
        String filePathName = fileToken.getPath() + fileToken.getFileName();
        File downloadedFile = new File(pathToDownloadFolder + Characters.SLASH + filePathName);
        downloadedFile.getParentFile().mkdirs();

        try (FileOutputStream fos = new FileOutputStream(downloadedFile)) {
            fos.write(keepWebApiClient.download(collectionUuid, filePathName));
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
        return downloadedFile;
    }

    public List<File> downloadFilesFromCollection(String collectionUuid, String pathToDownloadFolder) {

        // download requested collection and extract manifest text
        Collection requestedCollection = collectionsApiClient.get(collectionUuid);
        String manifestText = requestedCollection.getManifestText();

        // if directory with this collectionUUID does not exist - create one
        // if exists - abort (throw exception)
        File collectionTargetDir = setTargetDirectory(collectionUuid, pathToDownloadFolder);

        // decode manifest text and create list of ManifestStream objects containing KeepLocators and FileTokens
        List<ManifestStream> manifestStreams = manifestDecoder.decode(manifestText);

        //list of all downloaded files that will be returned by this method
        List<File> downloadedFilesFromCollection = new ArrayList<>();

        // download files for each manifest stream
        for (ManifestStream manifestStream : manifestStreams)
            downloadedFilesFromCollection.addAll(downloadFilesFromSingleManifestStream(manifestStream, collectionTargetDir));

        log.debug(String.format("Total of: %d files downloaded", downloadedFilesFromCollection.size()));
        return downloadedFilesFromCollection;
    }

    private File setTargetDirectory(String collectionUUID, String pathToDownloadFolder) {
        //local directory to save downloaded files
        File collectionTargetDir = new File(pathToDownloadFolder + Characters.SLASH + collectionUUID);
        if (collectionTargetDir.exists()) {
            throw new DownloadFolderAlreadyExistsException(String.format("Directory for collection UUID %s already exists", collectionUUID));
        } else {
            collectionTargetDir.mkdirs();
        }
        return collectionTargetDir;
    }

    private List<File> downloadFilesFromSingleManifestStream(ManifestStream manifestStream, File collectionTargetDir){
        List<File> downloadedFiles = new ArrayList<>();
        List<KeepLocator> keepLocators = manifestStream.getKeepLocators();
        DownloadHelper downloadHelper = new DownloadHelper(keepLocators);

        for (FileToken fileToken : manifestStream.getFileTokens()) {
            File downloadedFile = new File(collectionTargetDir.getAbsolutePath() + Characters.SLASH + fileToken.getFullPath()); //create file
            downloadedFile.getParentFile().mkdirs();

            try (FileOutputStream fos = new FileOutputStream(downloadedFile, true)) {
                downloadHelper.setBytesToDownload(fileToken.getFileSize()); //update file size info

                //this part needs to be repeated for each file until whole file is downloaded
                do {
                    downloadHelper.requestNewDataChunk(); //check if new data chunk needs to be downloaded
                    downloadHelper.writeDownFile(fos); // download data from chunk
                } while (downloadHelper.getBytesToDownload() != 0);

            } catch (IOException | ArvadosClientException e) {
                throw new ArvadosClientException(String.format("Unable to write down file %s", fileToken.getFileName()), e);
            }

            downloadedFiles.add(downloadedFile);
            log.debug(String.format("File %d / %d downloaded from manifest stream",
                    manifestStream.getFileTokens().indexOf(fileToken) + 1,
                    manifestStream.getFileTokens().size()));
        }
        return downloadedFiles;
    }

    private class DownloadHelper {

        // values for tracking file output streams and matching data chunks with initial files
        int currentDataChunkNumber;
        int bytesDownloadedFromChunk;
        long bytesToDownload;
        byte[] currentDataChunk;
        boolean remainingDataInChunk;
        final List<KeepLocator> keepLocators;

        private DownloadHelper(List<KeepLocator> keepLocators) {
            currentDataChunkNumber = -1;
            bytesDownloadedFromChunk = 0;
            remainingDataInChunk = false;
            this.keepLocators = keepLocators;
        }

        private long getBytesToDownload() {
            return bytesToDownload;
        }

        private void setBytesToDownload(long bytesToDownload) {
            this.bytesToDownload = bytesToDownload;
        }

        private void requestNewDataChunk() {
            if (!remainingDataInChunk) {
                currentDataChunkNumber++;
                if (currentDataChunkNumber < keepLocators.size()) {
                    //swap data chunk for next one
                    currentDataChunk = keepClient.getDataChunk(keepLocators.get(currentDataChunkNumber));
                    log.debug(String.format("%d of %d data chunks from manifest stream downloaded", currentDataChunkNumber + 1, keepLocators.size()));
                } else {
                    throw new ArvadosClientException("Data chunk required for download is missing.");
                }
            }
        }

        private void writeDownFile(FileOutputStream fos) throws IOException {
            //case 1: more bytes needed than available in current chunk (or whole current chunk needed) to download file
            if (bytesToDownload >= currentDataChunk.length - bytesDownloadedFromChunk) {
                writeDownWholeDataChunk(fos);
            }
            //case 2: current data chunk contains more bytes than is needed for this file
            else {
                writeDownDataChunkPartially(fos);
            }
        }

        private void writeDownWholeDataChunk(FileOutputStream fos) throws IOException {
            // write all remaining bytes from current chunk
            fos.write(currentDataChunk, bytesDownloadedFromChunk, currentDataChunk.length - bytesDownloadedFromChunk);
            //update bytesToDownload
            bytesToDownload -= (currentDataChunk.length - bytesDownloadedFromChunk);
            // set remaining data in chunk to false
            remainingDataInChunk = false;
            //reset bytesDownloadedFromChunk so that its set to 0 for the next chunk
            bytesDownloadedFromChunk = 0;
        }

        private void writeDownDataChunkPartially(FileOutputStream fos) throws IOException {
            //write all remaining bytes for this file from current chunk
            fos.write(currentDataChunk, bytesDownloadedFromChunk, (int) bytesToDownload);
            // update number of bytes downloaded from this chunk
            bytesDownloadedFromChunk += bytesToDownload;
            // set remaining data in chunk to true
            remainingDataInChunk = true;
            // reset bytesToDownload to exit while loop and move to the next file
            bytesToDownload = 0;
        }
    }
}