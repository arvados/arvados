// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { compose, Dispatch } from 'redux';
import { reduxForm, isPristine, isValid } from 'redux-form';
import { connect } from 'react-redux';
import { saveEditedUser } from 'store/user-profile/user-profile-actions';
import { UserProfilePanelRoot, UserProfilePanelRootDataProps } from 'views/user-profile-panel/user-profile-panel-root';
import { openDeactivateDialog, USER_PROFILE_FORM } from "store/user-profile/user-profile-actions";
import { matchUserProfileRoute } from 'routes/routes';
import { UserResource } from 'models/user';
import { getResource } from 'store/resources/resources';
import { openSetupShellAccount, loginAs } from 'store/users/users-actions';

const mapStateToProps = (state: RootState): UserProfilePanelRootDataProps => {
  const pathname = state.router.location ? state.router.location.pathname : '';
  const match = matchUserProfileRoute(pathname);
  const uuid = match ? match.params.id : state.auth.user?.uuid || '';
  // get user resource
  const user = getResource<UserResource>(uuid)(state.resources);
  // const subprocesses = getSubprocesses(uuid)(resources);

  return {

    isPristine: isPristine(USER_PROFILE_FORM)(state),
    isValid: isValid(USER_PROFILE_FORM)(state),
    initialValues: user,
    localCluster: state.auth.localCluster
}};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openSetupShellAccount: (uuid: string) => dispatch<any>(openSetupShellAccount(uuid)),
    loginAs: (uuid: string) => dispatch<any>(loginAs(uuid)),
    openDeactivateDialog: (uuid: string) => dispatch<any>(openDeactivateDialog(uuid)),
});

export const UserProfilePanel = compose(
    connect(mapStateToProps, mapDispatchToProps),
    reduxForm({
        form: USER_PROFILE_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(saveEditedUser(data));
        }
    }))(UserProfilePanelRoot);
