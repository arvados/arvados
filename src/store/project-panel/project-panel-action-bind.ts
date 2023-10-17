// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";

const PROJECT_PANEL_ID = "projectPanel";

export const projectPanelActions = bindDataExplorerActions(PROJECT_PANEL_ID);
