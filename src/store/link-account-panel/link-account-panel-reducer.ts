// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelActions, LinkAccountPanelAction } from "~/store/link-account-panel/link-account-panel-actions";
import { UserResource, User } from "~/models/user";

export interface LinkAccountPanelState {
    user: UserResource | undefined;
    userToken: string | undefined;
    userToLink: UserResource | undefined;
    userToLinkToken: string | undefined;
}

const initialState = {
    user: undefined,
    userToken: undefined,
    userToLink: undefined,
    userToLinkToken: undefined
};

export const linkAccountPanelReducer = (state: LinkAccountPanelState = initialState, action: LinkAccountPanelAction) =>
    linkAccountPanelActions.match(action, {
        default: () => state,
        LOAD_LINKING: ({ userToLink, user, userToken, userToLinkToken}) => ({ ...state, user, userToken, userToLink, userToLinkToken }),
        RESET_LINKING: () => ({ ...state, user: undefined, userToken: undefined, userToLink: undefined, userToLinkToken: undefined })
    });