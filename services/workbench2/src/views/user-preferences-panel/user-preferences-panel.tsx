// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { compose } from 'redux';
import { reduxForm, isPristine, isValid } from 'redux-form';
import { connect } from 'react-redux';
import { UserPreferencesPanelRoot, UserPreferencesPanelRootDataProps } from 'views/user-preferences-panel/user-preferences-panel-root';
import { USER_PREFERENCES_FORM, saveUserPreferences } from 'store/user-preferences/user-preferences-actions';

const mapStateToProps = (state: RootState): UserPreferencesPanelRootDataProps => {
    const uuid = state.auth.user?.uuid || '';

    return {
        isPristine: isPristine(USER_PREFERENCES_FORM)(state),
        isValid: isValid(USER_PREFERENCES_FORM)(state),
        userUuid: uuid,
        resources: state.resources,
    }
};

export const UserPreferencesPanel = compose(
    connect(mapStateToProps),
    reduxForm({
        form: USER_PREFERENCES_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(saveUserPreferences(data));
        }
    }))(UserPreferencesPanelRoot);
