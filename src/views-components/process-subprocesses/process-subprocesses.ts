// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { openProcessContextMenu } from '~/store/context-menu/context-menu-actions';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { ProcessSubprocessesDataProps, ProcessSubprocesses as SubprocessesComponent } from '~/views/process-panel/process-subprocesses';

type SubprocessesActionProps = Pick<ProcessSubprocessesDataProps, 'onContextMenu'>;

const mapStateToProps = (state: RootState) => ({
    // todo processPanel
    items: state.collectionPanel
});

const mapDispatchToProps = (dispatch: Dispatch): SubprocessesActionProps => ({
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => {
        dispatch<any>(openProcessContextMenu(event));       
    }
});

export const ProcessSubprocesses = connect(mapStateToProps, mapDispatchToProps)(SubprocessesComponent);