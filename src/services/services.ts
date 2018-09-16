// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "axios";
import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { CollectionService } from "./collection-service/collection-service";
import { TagService } from "./tag-service/tag-service";
import { CollectionFilesService } from "./collection-files-service/collection-files-service";
import { KeepService } from "./keep-service/keep-service";
import { WebDAV } from "~/common/webdav";
import { Config } from "~/common/config";
import { UserService } from './user-service/user-service';
import { AncestorService } from "~/services/ancestors-service/ancestors-service";
import { ResourceKind } from "~/models/resource";
import { ContainerRequestService } from './container-request-service/container-request-service';
import { ContainerService } from './container-service/container-service';
import { LogService } from './log-service/log-service';

export type ServiceRepository = ReturnType<typeof createServices>;

export const createServices = (config: Config, progressFn: (id: string, working: boolean) => void) => {
    const apiClient = Axios.create();
    apiClient.defaults.baseURL = config.baseUrl;

    const webdavClient = new WebDAV();
    webdavClient.defaults.baseURL = config.keepWebServiceUrl;

    const containerRequestService = new ContainerRequestService(apiClient, progressFn);
    const containerService = new ContainerService(apiClient, progressFn);
    const groupsService = new GroupsService(apiClient, progressFn);
    const keepService = new KeepService(apiClient, progressFn);
    const linkService = new LinkService(apiClient, progressFn);
    const logService = new LogService(apiClient, progressFn);
    const projectService = new ProjectService(apiClient, progressFn);
    const userService = new UserService(apiClient, progressFn);

    const ancestorsService = new AncestorService(groupsService, userService);
    const authService = new AuthService(apiClient, config.rootUrl, progressFn);
    const collectionService = new CollectionService(apiClient, webdavClient, authService, progressFn);
    const collectionFilesService = new CollectionFilesService(collectionService);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const tagService = new TagService(linkService);

    return {
        ancestorsService,
        apiClient,
        authService,
        collectionFilesService,
        collectionService,
        containerRequestService,
        containerService,
        favoriteService,
        groupsService,
        keepService,
        linkService,
        logService,
        projectService,
        tagService,
        userService,
        webdavClient,
    };
};

export const getResourceService = (kind?: ResourceKind) => (serviceRepository: ServiceRepository) => {
    switch (kind) {
        case ResourceKind.USER:
            return serviceRepository.userService;
        case ResourceKind.GROUP:
            return serviceRepository.groupsService;
        case ResourceKind.COLLECTION:
            return serviceRepository.collectionService;
        default:
            return undefined;
    }
};
