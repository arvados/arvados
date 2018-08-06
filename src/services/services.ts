// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { AxiosInstance } from "axios";
import { CollectionService } from "./collection-service/collection-service";
import Axios from "axios";
import { CollectionFilesService } from "./collection-files-service/collection-files-service";
import { KeepService } from "./keep-service/keep-service";

export interface ServiceRepository {
    apiClient: AxiosInstance;

    authService: AuthService;
    keepService: KeepService;
    groupsService: GroupsService;
    projectService: ProjectService;
    linkService: LinkService;
    favoriteService: FavoriteService;
    collectionService: CollectionService;
    collectionFilesService: CollectionFilesService;
}

export const createServices = (baseUrl: string): ServiceRepository => {
    const apiClient = Axios.create();
    apiClient.defaults.baseURL = `${baseUrl}/arvados/v1`;

    const authService = new AuthService(apiClient, baseUrl);
    const keepService = new KeepService(apiClient);
    const groupsService = new GroupsService(apiClient);
    const projectService = new ProjectService(apiClient);
    const linkService = new LinkService(apiClient);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const collectionService = new CollectionService(apiClient, keepService);
    const collectionFilesService = new CollectionFilesService(collectionService);

    return {
        apiClient,
        authService,
        keepService,
        groupsService,
        projectService,
        linkService,
        favoriteService,
        collectionService,
        collectionFilesService
    };
};
