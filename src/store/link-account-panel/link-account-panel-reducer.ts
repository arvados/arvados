// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelActions, LinkAccountPanelAction } from "~/store/link-account-panel/link-account-panel-actions";
import { UserResource } from "~/models/user";

export enum LinkAccountPanelStatus {
    INITIAL,
    LINKING,
    ERROR
}

export enum LinkAccountPanelError {
    NONE,
    NON_ADMIN,
    SAME_USER
}

export interface LinkAccountPanelState {
    user: UserResource | undefined;
    userToken: string | undefined;
    userToLink: UserResource | undefined;
    userToLinkToken: string | undefined;
    status: LinkAccountPanelStatus;
    error: LinkAccountPanelError;
}

const initialState = {
    user: undefined,
    userToken: undefined,
    userToLink: undefined,
    userToLinkToken: undefined,
    status: LinkAccountPanelStatus.INITIAL,
    error: LinkAccountPanelError.NONE
};

export const linkAccountPanelReducer = (state: LinkAccountPanelState = initialState, action: LinkAccountPanelAction) =>
    linkAccountPanelActions.match(action, {
        default: () => state,
        INIT: ({ user }) => ({
            ...state, user, state: LinkAccountPanelStatus.INITIAL, error: LinkAccountPanelError.NONE
        }),
        LOAD: ({ userToLink, user, userToken, userToLinkToken}) => ({
            ...state, user, userToken, userToLink, userToLinkToken, status: LinkAccountPanelStatus.LINKING, error: LinkAccountPanelError.NONE
        }),
        RESET: () => ({
            ...state, userToken: undefined, userToLink: undefined, userToLinkToken: undefined, status: LinkAccountPanelStatus.INITIAL,  error: LinkAccountPanelError.NONE
        }),
        INVALID: ({user, userToLink, error}) => ({
            ...state, user, userToLink, error, status: LinkAccountPanelStatus.ERROR
        })
    });