// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { CollectionService } from "./collection-service/collection-service";
import { TagService } from "./tag-service/tag-service";
import Axios from "axios";
import { CollectionFilesService } from "./collection-files-service/collection-files-service";
import { KeepService } from "./keep-service/keep-service";
import { WebDAV } from "../common/webdav";
import { Config } from "../common/config";

export type ServiceRepository = ReturnType<typeof createServices>;

export const createServices = (config: Config) => {
    const apiClient = Axios.create();
    apiClient.defaults.baseURL = `${config.apiHost}/arvados/v1`;

    const webdavClient = new WebDAV();
    webdavClient.defaults.baseURL = config.keepWebHost;

    const authService = new AuthService(apiClient, config.apiHost);
    const keepService = new KeepService(apiClient);
    const groupsService = new GroupsService(apiClient);
    const projectService = new ProjectService(apiClient);
    const linkService = new LinkService(apiClient);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const collectionService = new CollectionService(apiClient, keepService, webdavClient, authService);
    const tagService = new TagService(linkService);
    const collectionFilesService = new CollectionFilesService(collectionService);

    return {
        apiClient,
        webdavClient,
        authService,
        keepService,
        groupsService,
        projectService,
        linkService,
        favoriteService,
        collectionService,
        tagService,
        collectionFilesService
    };
};

