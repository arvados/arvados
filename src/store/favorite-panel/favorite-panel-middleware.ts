// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dataExplorerMiddleware } from "../data-explorer/data-explorer-middleware";
import { FavoritePanelMiddlewareService } from "./favorite-panel-middleware-service";

export const favoritePanelMiddleware = dataExplorerMiddleware(FavoritePanelMiddlewareService.getInstance());