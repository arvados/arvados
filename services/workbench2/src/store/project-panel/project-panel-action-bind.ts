// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";

// These are split into a separate file to avoid circular imports causing
// invariant violations with unit tests

export const PROJECT_PANEL_ID = "projectPanel";
export const projectPanelActions = bindDataExplorerActions(PROJECT_PANEL_ID);
