/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.facade;

import com.google.common.collect.Lists;
import org.arvados.client.api.client.CollectionsApiClient;
import org.arvados.client.api.client.GroupsApiClient;
import org.arvados.client.api.client.KeepWebApiClient;
import org.arvados.client.api.client.UsersApiClient;
import org.arvados.client.api.model.*;
import org.arvados.client.api.model.argument.Filter;
import org.arvados.client.api.model.argument.ListArgument;
import org.arvados.client.config.FileConfigProvider;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.logic.collection.FileToken;
import org.arvados.client.logic.collection.ManifestDecoder;
import org.arvados.client.logic.keep.FileDownloader;
import org.arvados.client.logic.keep.FileUploader;
import org.arvados.client.logic.keep.KeepClient;
import org.slf4j.Logger;

import java.io.File;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;

public class ArvadosFacade {

    private final ConfigProvider config;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(ArvadosFacade.class);
    private CollectionsApiClient collectionsApiClient;
    private GroupsApiClient groupsApiClient;
    private UsersApiClient usersApiClient;
    private FileDownloader fileDownloader;
    private FileUploader fileUploader;
    private static final String PROJECT = "project";
    private static final String SUBPROJECT = "sub-project";

    public ArvadosFacade(ConfigProvider config) {
        this.config = config;
        setFacadeFields();
    }

    public ArvadosFacade() {
        this.config = new FileConfigProvider();
        setFacadeFields();
    }

    private void setFacadeFields() {
        collectionsApiClient = new CollectionsApiClient(config);
        groupsApiClient = new GroupsApiClient(config);
        usersApiClient = new UsersApiClient(config);
        KeepClient keepClient = new KeepClient(config);
        ManifestDecoder manifestDecoder = new ManifestDecoder();
        KeepWebApiClient keepWebApiClient = new KeepWebApiClient(config);
        fileDownloader = new FileDownloader(keepClient, manifestDecoder, collectionsApiClient, keepWebApiClient);
        fileUploader = new FileUploader(keepClient, collectionsApiClient, config);
    }

    /**
     * This method downloads single file from collection using Arvados Keep-Web.
     * File is saved on a drive in specified location and returned.
     *
     * @param filePathName         path to the file in collection. If requested file is stored
     *                             directly in collection (not within its subdirectory) this
     *                             would be just the name of file (ex. 'file.txt').
     *                             Otherwise full file path must be passed (ex. 'folder/file.txt')
     * @param collectionUuid       uuid of collection containing requested file
     * @param pathToDownloadFolder path to location in which file should be saved.
     *                             Passed location must be a directory in which file of
     *                             that name does not already exist.
     * @return downloaded file
     */
    public File downloadFile(String filePathName, String collectionUuid, String pathToDownloadFolder) {
        return fileDownloader.downloadSingleFileUsingKeepWeb(filePathName, collectionUuid, pathToDownloadFolder);
    }

    /**
     * This method downloads all files from collection.
     * Directory named by collection uuid is created in specified location,
     * files are saved on a drive in this directory and list with downloaded
     * files is returned.
     *
     * @param collectionUuid       uuid of collection from which files are downloaded
     * @param pathToDownloadFolder path to location in which files should be saved.
     *                             New folder named by collection uuid, containing
     *                             downloaded files, is created in this location.
     *                             Passed location must be a directory in which folder
     *                             of that name does not already exist.
     * @param usingKeepWeb         if set to true files will be downloaded using Keep Web.
     *                             If set to false files will be downloaded using Keep Server API.
     * @return list containing downloaded files
     */
    public List<File> downloadCollectionFiles(String collectionUuid, String pathToDownloadFolder, boolean usingKeepWeb) {
        if (usingKeepWeb)
            return fileDownloader.downloadFilesFromCollectionUsingKeepWeb(collectionUuid, pathToDownloadFolder);
        return fileDownloader.downloadFilesFromCollection(collectionUuid, pathToDownloadFolder);
    }

    /**
     * Lists all FileTokens (objects containing information about files) for
     * specified collection.
     * Information in each FileToken includes file path, name, size and position
     * in data stream
     *
     * @param collectionUuid uuid of collection for which FileTokens are listed
     * @return list containing FileTokens for each file in specified collection
     */
    public List<FileToken> listFileInfoFromCollection(String collectionUuid) {
        return fileDownloader.listFileInfoFromCollection(collectionUuid);
    }

    /**
     * Creates and uploads new collection containing passed files.
     * Created collection has a default name and is uploaded to user's 'Home' project.
     *
     * @see ArvadosFacade#upload(List, String, String)
     * @param files    list of files to be uploaded within new collection
     * @return collection object mapped from JSON that is returned from server after successful upload
     */
    public Collection upload(List<File> files) {
        return upload(files, null, null);
    }

    /**
     * Creates and uploads new collection containing a single file.
     * Created collection has a default name and is uploaded to user's 'Home' project.
     *
     * @see ArvadosFacade#upload(List, String, String)
     * @param file file to be uploaded
     * @return collection object mapped from JSON that is returned from server after successful upload
     */
    public Collection upload(File file) {
        return upload(Collections.singletonList(file), null, null);
    }

    /**
     * Uploads new collection with specified name and containing selected files
     * to an existing project.
     *
     * @param sourceFiles    list of files to be uploaded within new collection
     * @param collectionName name for the newly created collection.
     *                       Collection with that name cannot be already created
     *                       in specified project. If null is passed
     *                       then collection name is set to default, containing
     *                       phrase 'New Collection' and a timestamp.
     * @param projectUuid    uuid of the project in which created collection is to be included.
     *                       If null is passed then collection is uploaded to user's 'Home' project.
     * @return collection object mapped from JSON that is returned from server after successful upload
     */
    public Collection upload(List<File> sourceFiles, String collectionName, String projectUuid) {
        return fileUploader.upload(sourceFiles, collectionName, projectUuid);
    }

    /**
     * Uploads a file to a specified collection.
     *
     * @see ArvadosFacade#uploadToExistingCollection(List, String)
     * @param file           file to be uploaded to existing collection. Filenames must be unique
     *                       in comparison with files already existing within collection.
     * @param collectionUUID UUID of collection to which files should be uploaded
     * @return collection object mapped from JSON that is returned from server after successful upload
     */
    public Collection uploadToExistingCollection(File file, String collectionUUID) {
        return fileUploader.uploadToExistingCollection(Collections.singletonList(file), collectionUUID);
    }

    /**
     * Uploads multiple files to an existing collection.
     *
     * @param files          list of files to be uploaded to existing collection.
     *                       File names must be unique - both within passed list and
     *                       in comparison with files already existing within collection.
     * @param collectionUUID UUID of collection to which files should be uploaded
     * @return collection object mapped from JSON that is returned from server after successful upload
     */
    public Collection uploadToExistingCollection(List<File> files, String collectionUUID) {
        return fileUploader.uploadToExistingCollection(files, collectionUUID);
    }

    /**
     * Creates and uploads new empty collection to specified project.
     *
     * @param collectionName name for the newly created collection.
     *                       Collection with that name cannot be already created
     *                       in specified project.
     * @param projectUuid    uuid of project that will contain uploaded empty collection.
     *                       To select home project pass current user's uuid from getCurrentUser()
     * @return collection object mapped from JSON that is returned from server after successful upload
     * @see ArvadosFacade#getCurrentUser()
     */
    public Collection createEmptyCollection(String collectionName, String projectUuid) {
        Collection collection = new Collection();
        collection.setOwnerUuid(projectUuid);
        collection.setName(collectionName);
        return collectionsApiClient.create(collection);
    }

    /**
     * Uploads multiple files to an existing collection.
     *
     * @param collectionUUID UUID of collection to which the files are to be copied
     * @param files          map of files to be copied to existing collection.
     *                       The map consists of a pair in the form of a filename and a filename
     *                       along with the Portable data hash
     * @return collection object mapped from JSON that is returned from server after successful copied
     */
    public Collection updateWithReplaceFiles(String collectionUUID, Map<String, String> files) {
        CollectionReplaceFiles replaceFilesRequest = new CollectionReplaceFiles();
        replaceFilesRequest.getReplaceFiles().putAll(files);
        return collectionsApiClient.update(collectionUUID, replaceFilesRequest);
    }

    /**
     * Returns current user information based on Api Token provided via configuration
     *
     * @return user object mapped from JSON that is returned from server based on provided Api Token.
     * It contains information about user who has this token assigned.
     */
    public User getCurrentUser() {
        return usersApiClient.current();
    }

    /**
     * Gets uuid of current user based on api Token provided in configuration and uses it to list all
     * projects that this user owns in Arvados.
     *
     * @return GroupList containing all groups that current user is owner of.
     * @see ArvadosFacade#getCurrentUser()
     */
    public GroupList showGroupsOwnedByCurrentUser() {
        ListArgument listArgument = ListArgument.builder()
                .filters(Arrays.asList(
                        Filter.of("owner_uuid", Filter.Operator.LIKE, getCurrentUser().getUuid()),
                        Filter.of("group_class", Filter.Operator.IN, Lists.newArrayList(PROJECT, SUBPROJECT)
                        )))
                .build();
        GroupList groupList = groupsApiClient.list(listArgument);
        log.debug("Groups owned by user:");
        groupList.getItems().forEach(m -> log.debug(m.getUuid() + " -- " + m.getName()));

        return groupList;
    }

    /**
     * Gets uuid of current user based on api Token provided in configuration and uses it to list all
     * projects that this user has read access to in Arvados.
     *
     * @return GroupList containing all groups that current user has read access to.
     */
    public GroupList showGroupsAccessibleByCurrentUser() {
        ListArgument listArgument = ListArgument.builder()
                .filters(Collections.singletonList(
                        Filter.of("group_class", Filter.Operator.IN, Lists.newArrayList(PROJECT, SUBPROJECT)
                        )))
                .build();
        GroupList groupList = groupsApiClient.list(listArgument);
        log.debug("Groups accessible by user:");
        groupList.getItems().forEach(m -> log.debug(m.getUuid() + " -- " + m.getName()));

        return groupList;
    }

    /**
     * Filters all collections from selected project and returns list of those that contain passed String in their name.
     * Operator "LIKE" is used so in order to obtain certain collection it is sufficient to pass just part of its name.
     * Returned collections in collectionList are ordered by date of creation (starting from oldest one).
     *
     * @param collectionName collections containing this param in their name will be returned.
     *                       Passing a wildcard is possible - for example passing "a%" searches for
     *                       all collections starting with "a".
     * @param projectUuid    uuid of project in which will be searched for collections with given name. To search home
     *                       project provide user uuid (from getCurrentUser())
     * @return object CollectionList containing all collections matching specified name criteria
     * @see ArvadosFacade#getCurrentUser()
     */
    public CollectionList getCollectionsFromProjectByName(String collectionName, String projectUuid) {
        ListArgument listArgument = ListArgument.builder()
                .filters(Arrays.asList(
                        Filter.of("owner_uuid", Filter.Operator.LIKE, projectUuid),
                        Filter.of("name", Filter.Operator.LIKE, collectionName)
                ))
                .order(Collections.singletonList("created_at"))
                .build();

        return collectionsApiClient.list(listArgument);
    }

    /**
     * Gets project details by uuid.
     *
     * @param projectUuid uuid of project
     * @return Group object containing information about project
     */
    public Group getProjectByUuid(String projectUuid) {
        Group project = groupsApiClient.get(projectUuid);
        log.debug("Retrieved " + project.getName() + " with UUID: " + project.getUuid());
        return project;
    }

    /**
     * Creates new project that will be a subproject of "home" for current user.
     *
     * @param projectName name for the newly created project
     * @return Group object containing information about created project
     * (mapped from JSON returned from server after creating the project)
     */
    public Group createNewProject(String projectName) {
        Group project = new Group();
        project.setName(projectName);
        project.setGroupClass(PROJECT);
        Group createdProject = groupsApiClient.create(project);
        log.debug("Project " + createdProject.getName() + " created with UUID: " + createdProject.getUuid());
        return createdProject;
    }

    /**
     * Deletes collection with specified uuid.
     *
     * @param collectionUuid uuid of collection to be deleted. User whose token is provided in configuration
     *                       must be authorized to delete such collection.
     * @return collection object with deleted collection (mapped from JSON returned from server after deleting the collection)
     */
    public Collection deleteCollection(String collectionUuid) {
        Collection deletedCollection = collectionsApiClient.delete(collectionUuid);
        log.debug("Collection: " + collectionUuid + " deleted.");
        return deletedCollection;
    }
}
