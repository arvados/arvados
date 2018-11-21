// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { SshKeyPanelRoot, SshKeyPanelRootDataProps, SshKeyPanelRootActionProps } from '~/views/ssh-key-panel/ssh-key-panel-root';
import { openSshKeyCreateDialog } from '~/store/auth/auth-action';

const mapStateToProps = (state: RootState): SshKeyPanelRootDataProps => {
    return {
        sshKeys: state.auth.sshKeys
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SshKeyPanelRootActionProps => ({
    onClick: () => {
        dispatch(openSshKeyCreateDialog());
    }
});

export const SshKeyPanel = connect(mapStateToProps, mapDispatchToProps)(SshKeyPanelRoot);