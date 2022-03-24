// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { RootState } from "store/store";
import { Dispatch } from 'redux';
import { initialize } from "redux-form";
import { ServiceRepository } from "services/services";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { propertiesActions } from 'store/properties/properties-actions';
import { getProperty } from 'store/properties/properties';
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { updateResources } from "store/resources/resources-actions";
import { dialogActions } from "store/dialog/dialog-actions";

export const USER_PROFILE_PANEL_ID = 'userProfilePanel';
export const USER_PROFILE_FORM = 'userProfileForm';
export const DEACTIVATE_DIALOG = 'deactivateDialog';
export const SETUP_DIALOG = 'setupDialog';

export const UserProfileGroupsActions = bindDataExplorerActions(USER_PROFILE_PANEL_ID);

export const getCurrentUserProfilePanelUuid = getProperty<string>(USER_PROFILE_PANEL_ID);

export const loadUserProfilePanel = (userUuid?: string) =>
  async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    // Get user uuid from route or use current user uuid
    const uuid = userUuid || getState().auth.user?.uuid;
    if (uuid) {
      await dispatch(propertiesActions.SET_PROPERTY({ key: USER_PROFILE_PANEL_ID, value: uuid }));
      const user = await services.userService.get(uuid);
      dispatch(initialize(USER_PROFILE_FORM, user));
      dispatch(updateResources([user]));
      dispatch(UserProfileGroupsActions.REQUEST_ITEMS());
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

export const openDeactivateDialog = (uuid: string) =>
  (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    dispatch(dialogActions.OPEN_DIALOG({
      id: DEACTIVATE_DIALOG,
      data: {
          title: 'Deactivate user',
          text: 'Are you sure you want to deactivate this user?',
          confirmButtonLabel: 'Deactvate',
          uuid
      }
  }));
}

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
}


export const setup = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const resources = await services.userService.setup(uuid);
            dispatch(updateResources(resources.items));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "User has been setup", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        } finally {
            dispatch(dialogActions.CLOSE_DIALOG({ id: SETUP_DIALOG }));
        }

    };

export const unsetup = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
          const user = await services.userService.unsetup(uuid);
          dispatch(updateResources([user]));
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
