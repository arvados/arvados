// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelActions, LinkAccountPanelAction } from "~/store/link-account-panel/link-account-panel-actions";
import { AccountToLink } from "~/models/link-account";

export interface LinkAccountPanelState {
    accountToLink: AccountToLink | undefined;
}

const initialState = {
    accountToLink: undefined
};

export const linkAccountPanelReducer = (state: LinkAccountPanelState = initialState, action: LinkAccountPanelAction) =>
    linkAccountPanelActions.match(action, {
        default: () => state,
        LOAD_LINKING: (accountToLink) => ({ ...state, accountToLink }),
        REMOVE_LINKING: () => ({...state, accountToLink: undefined})
    });