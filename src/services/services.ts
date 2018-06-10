// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import AuthService from "./auth-service/auth-service";
import ProjectService from "./project-service/project-service";

export const authService = new AuthService();
export const projectService = new ProjectService();
