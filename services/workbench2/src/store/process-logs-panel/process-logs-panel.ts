// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { matchProcessRoute } from 'routes/routes';
import { RouterState } from 'react-router-redux';

export interface ProcessLogsPanel {
    filters: string[];
    selectedFilter: string;
    logs: ProcessLogs;
}

export interface ProcessLogs {
    [logType: string]: {lastByte: number | undefined, contents: string[]};
}

export const getProcessPanelLogs = ({ selectedFilter, logs }: ProcessLogsPanel): string[] => {
    return logs[selectedFilter]?.contents || [];
};

export const getProcessLogsPanelCurrentUuid = (router: RouterState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    return match ? match.params.id : undefined;
};
