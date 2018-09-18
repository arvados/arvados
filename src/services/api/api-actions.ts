// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type ProgressFn = (id: string, working: boolean) => void;
export type ErrorFn = (id: string, message: string) => void;

export interface ApiActions {
    progressFn: ProgressFn;
    errorFn: ErrorFn;
}
