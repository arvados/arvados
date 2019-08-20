// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface RuntimeConstraints {
    ram: number;
    vcpus: number;
    keep_cache_ram?: number;
    API: boolean;
}
