// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { compose } from 'redux';
import { reduxForm, isPristine, isValid } from 'redux-form';
import { connect } from 'react-redux';
import { UserPreferencesPanelRoot, UserPreferencesPanelRootDataProps } from 'views/user-preferences-panel/user-preferences-panel-root';
import { USER_PROFILE_FORM } from "store/user-profile/user-profile-actions";

const mapStateToProps = (state: RootState): UserPreferencesPanelRootDataProps => {
    const uuid = state.auth.user?.uuid || '';

    return {
        isPristine: isPristine(USER_PROFILE_FORM)(state),
        isValid: isValid(USER_PROFILE_FORM)(state),
        userUuid: uuid,
        resources: state.resources,
    }
};

export const UserPreferencesPanel = compose(
    connect(mapStateToProps),
    reduxForm({
        form: USER_PROFILE_FORM,
        onSubmit: (data, dispatch) => {
            // dispatch(saveEditedUser(data));
        }
    }))(UserPreferencesPanelRoot);
