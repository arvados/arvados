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

export interface CUDAFeatures {
    driverVersion: string;
    hardwareCapability: string;
    deviceCount: number;
}

export interface NodeInstanceType {
    name: string;
    providerType: string;
    VCPUs: number;
    RAM: number;
    scratch: number;
    includedScratch: number;
    addedScratch: number;
    price: number;
    preemptible: boolean;
    CUDA: CUDAFeatures;
};

export interface NodeInfo {
    nodeInfo: NodeInstanceType | null;
};

export interface ProcessPanel {
    containerRequestUuid: string;
    filters: { [status: string]: boolean };
    inputRaw: WorkflowInputsData | null;
    inputParams: ProcessIOParameter[] | null;
    outputRaw: OutputDetails | null;
    outputDefinitions: CommandOutputParameter[];
    outputParams: ProcessIOParameter[] | null;
    nodeInfo: NodeInstanceType | null;
}

export const getProcessPanelCurrentUuid = (router: RouterState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    return match ? match.params.id : undefined;
};
