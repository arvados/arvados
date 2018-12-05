// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from '~/models/resource';

export interface NodeResource extends Resource {
    slotNumber: number;
    hostname: string;
    domain: string;
    ipAddress: string;
    jobUuid: string;
    firstPingAt: string;
    lastPingAt: string;
    status: string;
    info: NodeInfo;
    properties: NodeProperties;
}

export interface NodeInfo {
    lastAction: string;
    pingSecret: string;
    ec2InstanceId: string;
    slurmState?: string;
}

export interface NodeProperties {
    cloudNode: CloudNode;
    totalRamMb: number;
    totalCpuCores: number;
    totalScratchMb: number;
}

interface CloudNode {
    size: string;
    price: number;
}