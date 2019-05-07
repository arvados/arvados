// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelActions, LinkAccountPanelAction } from "~/store/link-account-panel/link-account-panel-actions";
import { UserResource } from "~/models/user";

export enum LinkAccountPanelStatus {
    INITIAL,
    HAS_SESSION_DATA,
    LINKING,
    ERROR
}

export enum LinkAccountPanelError {
    NONE,
    INACTIVE,
    NON_ADMIN,
    SAME_USER
}

export enum OriginatingUser {
    NONE,
    TARGET_USER,
    USER_TO_LINK
}

export interface LinkAccountPanelState {
    originatingUser: OriginatingUser | undefined;
    targetUser: UserResource | undefined;
    targetUserToken: string | undefined;
    userToLink: UserResource | undefined;
    userToLinkToken: string | undefined;
    status: LinkAccountPanelStatus;
    error: LinkAccountPanelError;
}

const initialState = {
    originatingUser: undefined,
    targetUser: undefined,
    targetUserToken: undefined,
    userToLink: undefined,
    userToLinkToken: undefined,
    status: LinkAccountPanelStatus.INITIAL,
    error: LinkAccountPanelError.NONE
};

export const linkAccountPanelReducer = (state: LinkAccountPanelState = initialState, action: LinkAccountPanelAction) =>
    linkAccountPanelActions.match(action, {
        default: () => state,
        LINK_INIT: ({ targetUser }) => ({
            ...state, targetUser, status: LinkAccountPanelStatus.INITIAL, error: LinkAccountPanelError.NONE, originatingUser: OriginatingUser.NONE
        }),
        LINK_LOAD: ({ originatingUser, userToLink, targetUser, targetUserToken, userToLinkToken}) => ({
            ...state, originatingUser, targetUser, targetUserToken, userToLink, userToLinkToken, status: LinkAccountPanelStatus.LINKING, error: LinkAccountPanelError.NONE
        }),
        LINK_INVALID: ({originatingUser, targetUser, userToLink, error}) => ({
            ...state, originatingUser, targetUser, userToLink, error, status: LinkAccountPanelStatus.ERROR
        }),
        HAS_SESSION_DATA: () => ({
            ...state, status: LinkAccountPanelStatus.HAS_SESSION_DATA
        })
    });