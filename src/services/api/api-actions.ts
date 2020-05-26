// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type ProgressFn = (id: string, working: boolean) => void;
export type ErrorFn = (id: string, error: any, showSnackBar?: boolean) => void;

export interface ApiActions {
    progressFn: ProgressFn;
    errorFn: ErrorFn;
}
