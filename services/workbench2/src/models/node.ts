// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from 'models/resource';

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
    last_action: string;
    ping_secret: string;
    ec2_instance_id: string;
    slurm_state?: string;
}

export interface NodeProperties {
    cloud_node: CloudNode;
    total_ram_mb: number;
    total_cpu_cores: number;
    total_scratch_mb: number;
}

interface CloudNode {
    size: string;
    price: number;
}