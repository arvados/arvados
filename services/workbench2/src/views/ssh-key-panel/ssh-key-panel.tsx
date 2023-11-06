// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { openSshKeyCreateDialog, openPublicKeyDialog } from 'store/auth/auth-action-ssh';
import { openSshKeyContextMenu } from 'store/context-menu/context-menu-actions';
import { SshKeyPanelRoot, SshKeyPanelRootDataProps, SshKeyPanelRootActionProps } from 'views/ssh-key-panel/ssh-key-panel-root';

const mapStateToProps = (state: RootState): SshKeyPanelRootDataProps => {
    const sshKeys = state.auth.sshKeys.filter((key) => {
      return key.authorizedUserUuid === (state.auth.user ? state.auth.user.uuid : null);
    });

    return {
        sshKeys: sshKeys,
        hasKeys: sshKeys!.length > 0
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SshKeyPanelRootActionProps => ({
    openSshKeyCreateDialog: () => {
        dispatch<any>(openSshKeyCreateDialog());
    },
    openRowOptions: (event, sshKey) => {
        dispatch<any>(openSshKeyContextMenu(event, sshKey));
    },
    openPublicKeyDialog: (name: string, publicKey: string) => {
        dispatch<any>(openPublicKeyDialog(name, publicKey));
    }
});

export const SshKeyPanel = connect(mapStateToProps, mapDispatchToProps)(SshKeyPanelRoot);
