// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface GPUParameters {
    stack: string;
    driver_version: string;
    hardware_target: string[];
    device_count: number;
    vram: number;
}

export interface RuntimeConstraints {
    ram: number;
    vcpus: number;
    keep_cache_ram?: number;
    keep_cache_disk?: number;
    API: boolean;
    gpu?: GPUParameters;
}
