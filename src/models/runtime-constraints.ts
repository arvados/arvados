// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface CUDAParameters {
    device_count: number;
    driver_version: string;
    hardware_capability: string;
}

export interface RuntimeConstraints {
    ram: number;
    vcpus: number;
    keep_cache_ram?: number;
    keep_cache_disk?: number;
    API: boolean;
    cuda?: CUDAParameters;
}
