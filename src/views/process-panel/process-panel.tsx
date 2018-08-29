// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RootState } from '~/store/store';
import { connect } from 'react-redux';
import { getProcess } from '~/store/processes/process';
import { Dispatch } from 'redux';
import { openProcessContextMenu } from '~/store/context-menu/context-menu-actions';
import { matchProcessRoute } from '~/routes/routes';
import { ProcessPanelRootDataProps, ProcessPanelRootActionProps, ProcessPanelRoot } from './process-panel-root';

const mapStateToProps = ({ router, resources }: RootState): ProcessPanelRootDataProps => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    const uuid = match ? match.params.id : '';
    return {
        process: getProcess(uuid)(resources)
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessPanelRootActionProps => ({
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => {
        dispatch<any>(openProcessContextMenu(event));
    }
});

export const ProcessPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessPanelRoot);
