// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { RootState } from "store/store";
import { Dispatch } from 'redux';
import { initialize, reset } from "redux-form";
import { ServiceRepository } from "services/services";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { propertiesActions } from 'store/properties/properties-actions';
import { getProperty } from 'store/properties/properties';
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { deleteResources, updateResources } from "store/resources/resources-actions";
import { dialogActions } from "store/dialog/dialog-actions";
import { filterResources } from "store/resources/resources";
import { ResourceKind } from "models/resource";
import { LinkClass, LinkResource } from "models/link";
import { BuiltinGroups, getBuiltinGroupUuid } from "models/group";

export const USER_PROFILE_PANEL_ID = 'userProfilePanel';
export const USER_PROFILE_FORM = 'userProfileForm';
export const DEACTIVATE_DIALOG = 'deactivateDialog';
export const SETUP_DIALOG = 'setupDialog';
export const ACTIVATE_DIALOG = 'activateDialog';
export const IS_PROFILE_INACCESSIBLE = 'isProfileInaccessible';

export const UserProfileGroupsActions = bindDataExplorerActions(USER_PROFILE_PANEL_ID);

export const getCurrentUserProfilePanelUuid = getProperty<string>(USER_PROFILE_PANEL_ID);
export const getUserProfileIsInaccessible = getProperty<boolean>(IS_PROFILE_INACCESSIBLE);

export const loadUserProfilePanel = (userUuid?: string) =>
  async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    // Reset isInacessible to ensure error screen is hidden
    dispatch(propertiesActions.SET_PROPERTY({ key: IS_PROFILE_INACCESSIBLE, value: false }));
    // Get user uuid from route or use current user uuid
    const uuid = userUuid || getState().auth.user?.uuid;
    if (uuid) {
      await dispatch(propertiesActions.SET_PROPERTY({ key: USER_PROFILE_PANEL_ID, value: uuid }));
      try {
        const user = await services.userService.get(uuid, false);
        dispatch(initialize(USER_PROFILE_FORM, user));
        dispatch(updateResources([user]));
        dispatch(UserProfileGroupsActions.REQUEST_ITEMS());
      } catch (e) {
        if (e.status === 404) {
          await dispatch(propertiesActions.SET_PROPERTY({ key: IS_PROFILE_INACCESSIBLE, value: true }));
          dispatch(reset(USER_PROFILE_FORM));
        } else {
          dispatch(snackbarActions.OPEN_SNACKBAR({
            message: 'Could not load user profile',
            kind: SnackbarKind.ERROR
          }));
        }
      }
    }
  }

export const saveEditedUser = (resource: any) =>
  async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
      try {
          const user = await services.userService.update(resource.uuid, resource);
          dispatch(updateResources([user]));
          dispatch(initialize(USER_PROFILE_FORM, user));
          dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Profile has been updated.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
      } catch (e) {
          dispatch(snackbarActions.OPEN_SNACKBAR({
              message: "Could not update profile",
              kind: SnackbarKind.ERROR,
          }));
      }
  };

export const openSetupDialog = (uuid: string) =>
  (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    dispatch(dialogActions.OPEN_DIALOG({
      id: SETUP_DIALOG,
      data: {
        title: 'Setup user',
        text: 'Are you sure you want to setup this user?',
        confirmButtonLabel: 'Confirm',
        uuid
      }
    }));
  };

export const openActivateDialog = (uuid: string) =>
  (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    dispatch(dialogActions.OPEN_DIALOG({
      id: ACTIVATE_DIALOG,
      data: {
        title: 'Activate user',
        text: 'Are you sure you want to activate this user?',
        confirmButtonLabel: 'Confirm',
        uuid
      }
    }));
  };

export const openDeactivateDialog = (uuid: string) =>
  (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    dispatch(dialogActions.OPEN_DIALOG({
      id: DEACTIVATE_DIALOG,
      data: {
        title: 'Deactivate user',
        text: 'Are you sure you want to deactivate this user?',
        confirmButtonLabel: 'Confirm',
        uuid
      }
    }));
  };

export const setup = (uuid: string) =>
  async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
      const resources = await services.userService.setup(uuid);
      dispatch(updateResources(resources.items));

      // Refresh data explorer
      dispatch(UserProfileGroupsActions.REQUEST_ITEMS());

      dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been setup", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    } catch (e) {
      dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
    } finally {
      dispatch(dialogActions.CLOSE_DIALOG({ id: SETUP_DIALOG }));
    }
  };

export const activate = (uuid: string) =>
  async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
      const user = await services.userService.activate(uuid);
      dispatch(updateResources([user]));

      // Refresh data explorer
      dispatch(UserProfileGroupsActions.REQUEST_ITEMS());

      dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been activated", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    } catch (e) {
      dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
    }
  };

export const deactivate = (uuid: string) =>
  async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
      const { resources, auth } = getState();
      // Call unsetup
      const user = await services.userService.unsetup(uuid);
      dispatch(updateResources([user]));

      // Find and remove all users membership
      const allUsersGroupUuid = getBuiltinGroupUuid(auth.localCluster, BuiltinGroups.ALL);
      const memberships = filterResources((resource: LinkResource) =>
          resource.kind === ResourceKind.LINK &&
          resource.linkClass === LinkClass.PERMISSION &&
          resource.headUuid === allUsersGroupUuid &&
          resource.tailUuid === uuid
      )(resources);
      // Remove all users membership locally
      dispatch<any>(deleteResources(memberships.map(link => link.uuid)));

      // Refresh data explorer
      dispatch(UserProfileGroupsActions.REQUEST_ITEMS());

      dispatch(snackbarActions.OPEN_SNACKBAR({
        message: "User has been deactivated.",
        hideDuration: 2000,
        kind: SnackbarKind.SUCCESS
      }));
    } catch (e) {
      dispatch(snackbarActions.OPEN_SNACKBAR({
        message: "Could not deactivate user",
        kind: SnackbarKind.ERROR,
      }));
    }
  };
