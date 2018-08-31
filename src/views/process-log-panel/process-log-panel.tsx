// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RootState } from '~/store/store';
import { connect } from 'react-redux';
import { getProcess } from '~/store/processes/process';
import { Dispatch } from 'redux';
import { openProcessContextMenu } from '~/store/context-menu/context-menu-actions';
import { matchProcessLogRoute } from '~/routes/routes';
import { ProcessLogPanelRootDataProps, ProcessLogPanelRootActionProps, ProcessLogPanelRoot } from './process-log-panel-root';

const SELECT_OPTIONS = [
    { label: 'Dispatch', value: 'dispatch' },
    { label: 'Crunch-run', value: 'crunch-run' },
    { label: 'Crunchstat', value: 'crunchstat' },
    { label: 'Hoststat', value: 'hoststat' },
    { label: 'Node-info', value: 'node-info' },
    { label: 'Arv-mount', value: 'arv-mount' },
    { label: 'Stdout', value: 'stdout' },
    { label: 'Stderr', value: 'stderr' }
];

export interface Log {
    object_uuid: string;
    event_at: string;
    event_type: string;
    summary: string;
    properties: any;
}

export interface FilterOption {
    label: string;
    value: string;
}

const mapStateToProps = ({ router, resources }: RootState): ProcessLogPanelRootDataProps => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessLogRoute(pathname);
    const uuid = match ? match.params.id : '';
    return {
        process: getProcess(uuid)(resources),
        selectedFilter: SELECT_OPTIONS[0],
        filters: SELECT_OPTIONS
        // lines: string[]
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessLogPanelRootActionProps => ({
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => {
        dispatch<any>(openProcessContextMenu(event));
    },
    onChange: (filter: FilterOption) => { return; }
});

export const ProcessLogPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessLogPanelRoot);
