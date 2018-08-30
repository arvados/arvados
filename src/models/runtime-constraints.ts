// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface RuntimeConstraints {
    ram: number;
    vcpus: number;
    keepCacheRam: number;
    API: boolean;
}
