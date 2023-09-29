// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';

export const COLLECTIONS_CONTENT_ADDRESS_PANEL_ID = 'collectionsContentAddressPanel';

export const collectionsContentAddressActions = bindDataExplorerActions(COLLECTIONS_CONTENT_ADDRESS_PANEL_ID);

export const loadCollectionsContentAddressPanel = () =>
    (dispatch: Dispatch) => {
        dispatch(collectionsContentAddressActions.REQUEST_ITEMS());
    };
