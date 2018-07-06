// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import AuthService from "./auth-service/auth-service";
import CollectionService from "./collection-service/collection-service";
import GroupsService from "./groups-service/groups-service";
import { serverApi } from "../common/api/server-api";
import ProjectService from "./project-service/project-service";

export const authService = new AuthService();
export const collectionService = new CollectionService();
export const groupsService = new GroupsService(serverApi);
export const projectService = new ProjectService(serverApi);
