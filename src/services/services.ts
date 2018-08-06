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
import { Resource } from "../models/resource";
import { CollectionService } from "./collection-service/collection-service";
import { TagService } from "./tag-service/tag-service";
import Axios from "axios";

export interface ServiceRepository {
    apiClient: AxiosInstance;

    authService: AuthService;
    groupsService: GroupsService;
    projectService: ProjectService;
    linkService: LinkService;
    favoriteService: FavoriteService;
    collectionService: CommonResourceService<Resource>;
    tagService: TagService;
}

export const createServices = (baseUrl: string): ServiceRepository => {
    const apiClient = Axios.create();
    apiClient.defaults.baseURL = `${baseUrl}/arvados/v1`;

    const authService = new AuthService(apiClient, baseUrl);
    const groupsService = new GroupsService(apiClient);
    const projectService = new ProjectService(apiClient);
    const linkService = new LinkService(apiClient);
    const favoriteService = new FavoriteService(linkService, groupsService);
    const collectionService = new CollectionService(apiClient);
    const tagService = new TagService(linkService);
    
    return {
        apiClient,
        authService,
        groupsService,
        projectService,
        linkService,
        favoriteService,
        collectionService,
        tagService
    };
};
