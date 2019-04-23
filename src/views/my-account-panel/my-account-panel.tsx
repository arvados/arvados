// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { compose } from 'redux';
import { reduxForm, isPristine, isValid } from 'redux-form';
import { connect } from 'react-redux';
import { saveEditedUser, openLinkAccount } from '~/store/my-account/my-account-panel-actions';
import { MyAccountPanelRoot, MyAccountPanelRootDataProps, MyAccountPanelRootActionProps } from '~/views/my-account-panel/my-account-panel-root';
import { MY_ACCOUNT_FORM } from "~/store/my-account/my-account-panel-actions";

const mapStateToProps = (state: RootState): MyAccountPanelRootDataProps => ({
    isPristine: isPristine(MY_ACCOUNT_FORM)(state),
    isValid: isValid(MY_ACCOUNT_FORM)(state),
    initialValues: state.auth.user,
    localCluster: state.auth.localCluster
});

const mapDispatchToProps = (dispatch: Dispatch): MyAccountPanelRootActionProps => ({
    openLinkAccount: () => dispatch<any>(openLinkAccount())
});

export const MyAccountPanel = compose(
    connect(mapStateToProps, mapDispatchToProps),
    reduxForm({
        form: MY_ACCOUNT_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(saveEditedUser(data));
        }
    }))(MyAccountPanelRoot);
