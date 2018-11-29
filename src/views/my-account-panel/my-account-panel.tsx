// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { MyAccountPanelRoot, MyAccountPanelRootDataProps, MyAccountPanelRootActionProps } from '~/views/my-account-panel/my-account-panel-root';

const mapStateToProps = (state: RootState): MyAccountPanelRootDataProps => ({
    user: state.auth.user
});

const mapDispatchToProps = (dispatch: Dispatch): MyAccountPanelRootActionProps => ({

});

export const MyAccountPanel = connect(mapStateToProps, mapDispatchToProps)(MyAccountPanelRoot);