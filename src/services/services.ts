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
import { ApiActions } from "~/services/api/api-actions";
import { WorkflowService } from "~/services/workflow-service/workflow-service";
import { SearchService } from '~/services/search-service/search-service';
import { PermissionService } from "~/services/permission-service/permission-service";

export type ServiceRepository = ReturnType<typeof createServices>;

export const createServices = (config: Config, actions: ApiActions) => {
    const apiClient = Axios.create();
    apiClient.defaults.baseURL = config.baseUrl;

    const webdavClient = new WebDAV();
    webdavClient.defaults.baseURL = config.keepWebServiceUrl;

    const containerRequestService = new ContainerRequestService(apiClient, actions);
    const containerService = new ContainerService(apiClient, actions);
    const groupsService = new GroupsService(apiClient, actions);
    const keepService = new KeepService(apiClient, actions);
    const linkService = new LinkService(apiClient, actions);
    const logService = new LogService(apiClient, actions);
    const permissionService = new PermissionService(apiClient, actions);
    const projectService = new ProjectService(apiClient, actions);
    const userService = new UserService(apiClient, actions);
    const workflowService = new WorkflowService(apiClient, actions);

    const ancestorsService = new AncestorService(groupsService, userService);
    const authService = new AuthService(apiClient, config.rootUrl, actions);
    const collectionService = new CollectionService(apiClient, webdavClient, authService, actions);
    const collectionFilesService = new CollectionFilesService(collectionService);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const tagService = new TagService(linkService);
    const searchService = new SearchService();

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
        permissionService,
        projectService,
        searchService,
        tagService,
        userService,
        webdavClient,
        workflowService,
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
