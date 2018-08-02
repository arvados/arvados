// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { AxiosInstance } from "axios";
import { CommonResourceService } from "../common/api/common-resource-service";
import { CollectionResource } from "../models/collection";
import { Resource } from "../models/resource";
import Axios from "axios";

export interface ServiceRepository {
    apiClient: AxiosInstance;
    authClient: AxiosInstance;

    authService: AuthService;
    groupsService: GroupsService;
    projectService: ProjectService;
    linkService: LinkService;
    favoriteService: FavoriteService;
    collectionService: CommonResourceService<Resource>;
}

export const createServices = (baseUrl: string): ServiceRepository => {
    const authClient = Axios.create();
    const apiClient = Axios.create();

    authClient.defaults.baseURL = baseUrl;
    apiClient.defaults.baseURL = `${baseUrl}/arvados/v1`;

    const authService = new AuthService(authClient, apiClient);
    const groupsService = new GroupsService(apiClient);
    const projectService = new ProjectService(apiClient);
    const linkService = new LinkService(apiClient);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const collectionService = new CommonResourceService<CollectionResource>(apiClient, "collections");

    return {
        apiClient,
        authClient,
        authService,
        groupsService,
        projectService,
        linkService,
        favoriteService,
        collectionService
    };
};
