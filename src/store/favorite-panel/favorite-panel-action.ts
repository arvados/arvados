// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";
import { FavoritePanelMiddlewareService } from "./favorite-panel-middleware-service";

export const favoritePanelActions = bindDataExplorerActions(FavoritePanelMiddlewareService.getInstance());
