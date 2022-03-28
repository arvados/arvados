// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { compose, Dispatch } from 'redux';
import { reduxForm, isPristine, isValid } from 'redux-form';
import { connect } from 'react-redux';
import { UserResource } from 'models/user';
import { saveEditedUser } from 'store/user-profile/user-profile-actions';
import { UserProfilePanelRoot, UserProfilePanelRootDataProps } from 'views/user-profile-panel/user-profile-panel-root';
import { USER_PROFILE_FORM } from "store/user-profile/user-profile-actions";
import { matchUserProfileRoute } from 'routes/routes';
import { openUserContextMenu } from 'store/context-menu/context-menu-actions';

const mapStateToProps = (state: RootState): UserProfilePanelRootDataProps => {
  const pathname = state.router.location ? state.router.location.pathname : '';
  const match = matchUserProfileRoute(pathname);
  const uuid = match ? match.params.id : state.auth.user?.uuid || '';

  return {
    isAdmin: state.auth.user!.isAdmin,
    isSelf: state.auth.user!.uuid === uuid,
    isPristine: isPristine(USER_PROFILE_FORM)(state),
    isValid: isValid(USER_PROFILE_FORM)(state),
    localCluster: state.auth.localCluster,
    userUuid: uuid,
    resources: state.resources,
}};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleContextMenu: (event, resource: UserResource) => dispatch<any>(openUserContextMenu(event, resource)),
});

export const UserProfilePanel = compose(
    connect(mapStateToProps, mapDispatchToProps),
    reduxForm({
        form: USER_PROFILE_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(saveEditedUser(data));
        }
    }))(UserProfilePanelRoot);
