// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const ALL_PROCESSES_PANEL_ID = "allProcessesPanel";
export const allProcessesPanelActions = bindDataExplorerActions(ALL_PROCESSES_PANEL_ID);

export const loadAllProcessesPanel = () => allProcessesPanelActions.REQUEST_ITEMS();
