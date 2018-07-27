// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { authClient, apiClient } from "../common/api/server-api";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";
import { CollectionCreationService } from "./collection-service/collection-service";

export const authService = new AuthService(authClient, apiClient);
export const groupsService = new GroupsService(apiClient);
export const projectService = new ProjectService(apiClient);
export const collectionCreationService = new CollectionCreationService(apiClient);
export const linkService = new LinkService(apiClient);
export const favoriteService = new FavoriteService(linkService, groupsService);
