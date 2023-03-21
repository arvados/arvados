// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "axios";
import { AxiosInstance } from "axios";
import { ApiClientAuthorizationService } from 'services/api-client-authorization-service/api-client-authorization-service';
import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { CollectionService } from "./collection-service/collection-service";
import { TagService } from "./tag-service/tag-service";
import { KeepService } from "./keep-service/keep-service";
import { WebDAV } from "common/webdav";
import { Config } from "common/config";
import { UserService } from './user-service/user-service';
import { AncestorService } from "services/ancestors-service/ancestors-service";
import { ResourceKind } from "models/resource";
import { ContainerRequestService } from './container-request-service/container-request-service';
import { ContainerService } from './container-service/container-service';
import { LogService } from './log-service/log-service';
import { ApiActions } from "services/api/api-actions";
import { WorkflowService } from "services/workflow-service/workflow-service";
import { SearchService } from 'services/search-service/search-service';
import { PermissionService } from "services/permission-service/permission-service";
import { VirtualMachinesService } from "services/virtual-machines-service/virtual-machines-service";
import { RepositoriesService } from 'services/repositories-service/repositories-service';
import { AuthorizedKeysService } from 'services/authorized-keys-service/authorized-keys-service';
import { VocabularyService } from 'services/vocabulary-service/vocabulary-service';
import { FileViewersConfigService } from 'services/file-viewers-config-service/file-viewers-config-service';
import { LinkAccountService } from "./link-account-service/link-account-service";
import parse from "parse-duration";

export type ServiceRepository = ReturnType<typeof createServices>;

export function setAuthorizationHeader(services: ServiceRepository, token: string) {
    services.apiClient.defaults.headers.common = {
        Authorization: `Bearer ${token}`
    };
    services.webdavClient.setAuthorization(`Bearer ${token}`);
}

export function removeAuthorizationHeader(services: ServiceRepository) {
    delete services.apiClient.defaults.headers.common;
    services.webdavClient.setAuthorization(undefined);
}

export const createServices = (config: Config, actions: ApiActions, useApiClient?: AxiosInstance) => {
    // Need to give empty 'headers' object or it will create an
    // instance with a reference to the global default headers object,
    // which is very bad because that means setAuthorizationHeader
    // would update the global default instead of the instance default.
    const apiClient = useApiClient || Axios.create({ headers: {} });
    apiClient.defaults.baseURL = config.baseUrl;

    const webdavClient = new WebDAV({
        baseURL: config.keepWebServiceUrl
    });

    const apiClientAuthorizationService = new ApiClientAuthorizationService(apiClient, actions);
    const authorizedKeysService = new AuthorizedKeysService(apiClient, actions);
    const containerRequestService = new ContainerRequestService(apiClient, actions);
    const containerService = new ContainerService(apiClient, actions);
    const groupsService = new GroupsService(apiClient, actions);
    const keepService = new KeepService(apiClient, actions);
    const linkService = new LinkService(apiClient, actions);
    const logService = new LogService(apiClient, actions);
    const permissionService = new PermissionService(apiClient, actions);
    const projectService = new ProjectService(apiClient, actions);
    const repositoriesService = new RepositoriesService(apiClient, actions);
    const userService = new UserService(apiClient, actions);
    const virtualMachineService = new VirtualMachinesService(apiClient, actions);
    const workflowService = new WorkflowService(apiClient, actions);
    const linkAccountService = new LinkAccountService(apiClient, actions);

    const ancestorsService = new AncestorService(groupsService, userService);

    const idleTimeout = (config && config.clusterConfig && config.clusterConfig.Workbench.IdleTimeout) || '0s';
    const authService = new AuthService(apiClient, config.rootUrl, actions,
        (parse(idleTimeout, 's') || 0) > 0);

    const collectionService = new CollectionService(apiClient, webdavClient, authService, actions);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const tagService = new TagService(linkService);
    const searchService = new SearchService();
    const vocabularyService = new VocabularyService(config.vocabularyUrl);
    const fileViewersConfig = new FileViewersConfigService(config.fileViewersConfigUrl);

    return {
        ancestorsService,
        apiClient,
        apiClientAuthorizationService,
        authService,
        authorizedKeysService,
        collectionService,
        containerRequestService,
        containerService,
        favoriteService,
        fileViewersConfig,
        groupsService,
        keepService,
        linkService,
        logService,
        permissionService,
        projectService,
        repositoriesService,
        searchService,
        tagService,
        userService,
        virtualMachineService,
        webdavClient,
        workflowService,
        vocabularyService,
        linkAccountService
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
