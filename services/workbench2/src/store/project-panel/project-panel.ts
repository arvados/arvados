// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getProperty } from "store/properties/properties";
import { RootState } from "store/store";

export const PROJECT_PANEL_CURRENT_UUID = "projectPanelCurrentUuid";
export const IS_PROJECT_PANEL_TRASHED = "isProjectPanelTrashed";

export const getProjectPanelCurrentUuid = (state: RootState) => getProperty<string>(PROJECT_PANEL_CURRENT_UUID)(state.properties);
