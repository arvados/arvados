// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AuthService } from "./auth-service/auth-service";
import { GroupsService } from "./groups-service/groups-service";
import { serverApi } from "../common/api/server-api";
import { ProjectService } from "./project-service/project-service";
import { LinkService } from "./link-service/link-service";
import { FavoriteService } from "./favorite-service/favorite-service";

export const authService = new AuthService(serverApi);
export const groupsService = new GroupsService(serverApi);
export const projectService = new ProjectService(serverApi);
export const linkService = new LinkService(serverApi);
export const favoriteService = new FavoriteService(linkService, groupsService);
