// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch, compose } from 'redux';
import { reduxForm, reset } from 'redux-form';
import { connect } from 'react-redux';
import { MyAccountPanelRoot, MyAccountPanelRootDataProps, MyAccountPanelRootActionProps, MY_ACCOUNT_FORM } from '~/views/my-account-panel/my-account-panel-root';

const mapStateToProps = (state: RootState): MyAccountPanelRootDataProps => ({
    user: state.auth.user
});

const mapDispatchToProps = (dispatch: Dispatch): MyAccountPanelRootActionProps => ({

});

export const MyAccountPanel = compose(connect(mapStateToProps, mapDispatchToProps), reduxForm({
    form: MY_ACCOUNT_FORM,
    onSubmit: (data, dispatch) => {
        // dispatch(moveProject(data));

    }
}))(MyAccountPanelRoot);