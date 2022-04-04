// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from "store/store";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { getUserAccountStatus, UserAccountStatus } from "store/users/users-actions";
import { matchMyAccountRoute, matchUserProfileRoute } from "routes/routes";

export const isAdmin = (state: RootState, resource: ContextMenuResource) => {
  return state.auth.user!.isAdmin;
}

export const canActivateUser = (state: RootState, resource: ContextMenuResource) => {
  const status = getUserAccountStatus(state, resource.uuid);
  return status === UserAccountStatus.INACTIVE ||
    status === UserAccountStatus.SETUP;
};

export const canDeactivateUser = (state: RootState, resource: ContextMenuResource) => {
  const status = getUserAccountStatus(state, resource.uuid);
  return status === UserAccountStatus.SETUP ||
    status === UserAccountStatus.ACTIVE;
};

export const canSetupUser = (state: RootState, resource: ContextMenuResource) => {
  const status = getUserAccountStatus(state, resource.uuid);
  return status === UserAccountStatus.INACTIVE;
};

export const needsUserProfileLink = (state: RootState, resource: ContextMenuResource) => (
  state.router.location ?
    !(matchUserProfileRoute(state.router.location.pathname)
      || matchMyAccountRoute(state.router.location.pathname)
    ) : true
);

export const isOtherUser = (state: RootState, resource: ContextMenuResource) => {
  return state.auth.user!.uuid !== resource.uuid;
};
