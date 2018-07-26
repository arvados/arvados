// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dataExplorerMiddleware } from "../data-explorer/data-explorer-middleware";
import { ProjectPanelMiddlewareService } from "./project-panel-middleware-service";

export const projectPanelMiddleware = dataExplorerMiddleware(ProjectPanelMiddlewareService.getInstance());
