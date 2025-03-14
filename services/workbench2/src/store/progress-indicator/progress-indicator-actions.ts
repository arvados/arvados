// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";

export const progressIndicatorActions = unionize({
    START_WORKING: ofType<string>(),
    STOP_WORKING: ofType<string>(),
});

export type ProgressIndicatorAction = UnionOf<typeof progressIndicatorActions>;

export const WORKBENCH_LOADING_SCREEN = "workbenchLoadingScreen";

