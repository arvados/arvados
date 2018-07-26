// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";
import { ProjectPanelMiddlewareService } from "./project-panel-middleware-service";

export const projectPanelActions = bindDataExplorerActions(ProjectPanelMiddlewareService.getInstance());
