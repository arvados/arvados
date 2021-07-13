// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";
import { Dispatch } from 'redux';
import { ServiceRepository } from "services/services";
import { RootState } from 'store/store';

export const SHARED_WITH_ME_PANEL_ID = "sharedWithMePanel";
export const sharedWithMePanelActions = bindDataExplorerActions(SHARED_WITH_ME_PANEL_ID);

export const loadSharedWithMePanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(sharedWithMePanelActions.REQUEST_ITEMS());
    };


