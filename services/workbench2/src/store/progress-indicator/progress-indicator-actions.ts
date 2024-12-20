// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";

export const progressIndicatorsActions = unionize({
    START_WORKING: ofType<string>(),
    STOP_WORKING: ofType<string>(),
});

export type ProgressIndicatorsAction = UnionOf<typeof progressIndicatorsActions>;

export const WORKBENCH_LOADING_SCREEN = "workbenchLoadingScreen";

