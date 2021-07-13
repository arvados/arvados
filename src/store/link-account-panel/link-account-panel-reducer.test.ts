// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { linkAccountPanelReducer, LinkAccountPanelError, LinkAccountPanelStatus, OriginatingUser } from "store/link-account-panel/link-account-panel-reducer";
import { linkAccountPanelActions } from "store/link-account-panel/link-account-panel-actions";

describe('link-account-panel-reducer', () => {
    const initialState = undefined;

    it('handles initial link account state', () => {
        const targetUser = { } as any;
        targetUser.username = "targetUser";

        const state = linkAccountPanelReducer(initialState, linkAccountPanelActions.LINK_INIT({targetUser}));
        expect(state).toEqual({
            targetUser,
            isProcessing: false,
            selectedCluster: undefined,
            targetUserToken: undefined,
            userToLink: undefined,
            userToLinkToken: undefined,
            originatingUser: OriginatingUser.NONE,
            error: LinkAccountPanelError.NONE,
            status: LinkAccountPanelStatus.INITIAL
        });
    });

    it('handles loaded link account state', () => {
        const targetUser = { } as any;
        targetUser.username = "targetUser";
        const targetUserToken = "targettoken";

        const userToLink = { } as any;
        userToLink.username = "userToLink";
        const userToLinkToken = "usertoken";

        const originatingUser = OriginatingUser.TARGET_USER;

        const state = linkAccountPanelReducer(initialState, linkAccountPanelActions.LINK_LOAD({
            originatingUser, targetUser, targetUserToken, userToLink, userToLinkToken}));
        expect(state).toEqual({
            targetUser,
            targetUserToken,
            isProcessing: false,
            selectedCluster: undefined,
            userToLink,
            userToLinkToken,
            originatingUser: OriginatingUser.TARGET_USER,
            error: LinkAccountPanelError.NONE,
            status: LinkAccountPanelStatus.LINKING
        });
    });

    it('handles loaded invalid account state', () => {
        const targetUser = { } as any;
        targetUser.username = "targetUser";

        const userToLink = { } as any;
        userToLink.username = "userToLink";

        const originatingUser = OriginatingUser.TARGET_USER;
        const error = LinkAccountPanelError.NON_ADMIN;

        const state = linkAccountPanelReducer(initialState, linkAccountPanelActions.LINK_INVALID({targetUser, userToLink, originatingUser, error}));
        expect(state).toEqual({
            targetUser,
            targetUserToken: undefined,
            isProcessing: false,
            selectedCluster: undefined,
            userToLink,
            userToLinkToken: undefined,
            originatingUser: OriginatingUser.TARGET_USER,
            error: LinkAccountPanelError.NON_ADMIN,
            status: LinkAccountPanelStatus.ERROR
        });
    });
});
