// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { Dispatch } from 'redux';
import { propertiesActions } from '~/store/properties/properties-actions';
import { getProperty } from '~/store/properties/properties';

export const GROUP_DETAILS_PANEL_ID = 'groupDetailsPanel';

export const GroupDetailsPanelActions = bindDataExplorerActions(GROUP_DETAILS_PANEL_ID);

export const loadGroupDetailsPanel = (groupUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(propertiesActions.SET_PROPERTY({ key: GROUP_DETAILS_PANEL_ID, value: groupUuid }));
        dispatch(GroupDetailsPanelActions.REQUEST_ITEMS());
    };

export const getCurrentGroupDetailsPanelUuid = getProperty<string>(GROUP_DETAILS_PANEL_ID);
