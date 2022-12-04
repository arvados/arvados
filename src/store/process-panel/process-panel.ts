// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { WorkflowInputsData } from 'models/workflow';
import { RouterState } from "react-router-redux";
import { matchProcessRoute } from "routes/routes";
import { ProcessIOParameter } from "views/process-panel/process-io-card";
import { CommandOutputParameter } from 'cwlts/mappings/v1.0/CommandOutputParameter';

export type OutputDetails = {
    rawOutputs?: any;
    pdh?: string;
}

export interface ProcessPanel {
    containerRequestUuid: string;
    filters: { [status: string]: boolean };
    inputRaw: WorkflowInputsData | null;
    inputParams: ProcessIOParameter[] | null;
    outputRaw: OutputDetails | null;
    outputDefinitions: CommandOutputParameter[];
    outputParams: ProcessIOParameter[] | null;
}

export const getProcessPanelCurrentUuid = (router: RouterState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    return match ? match.params.id : undefined;
};
