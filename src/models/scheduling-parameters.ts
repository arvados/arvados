// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface SchedulingParameters {
    partitions?: string[];
    preemptible?: boolean;
    max_run_time?: number;
}
