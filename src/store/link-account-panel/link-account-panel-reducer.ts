// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelActions, LinkAccountPanelAction } from "~/store/link-account-panel/link-account-panel-actions";
import { UserResource, User } from "~/models/user";

export interface LinkAccountPanelState {
    user: UserResource | undefined;
    userToLink: UserResource | undefined;
}

const initialState = {
    user: undefined,
    userToLink: undefined
};

export const linkAccountPanelReducer = (state: LinkAccountPanelState = initialState, action: LinkAccountPanelAction) =>
    linkAccountPanelActions.match(action, {
        default: () => state,
        LOAD_LINKING: ({ userToLink, user }) => ({ ...state, user, userToLink }),
        REMOVE_LINKING: () => ({ ...state, user: undefined, userToLink: undefined })
    });