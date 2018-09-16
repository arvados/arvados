// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export const progressIndicatorActions = unionize({
    START: ofType<string>(),
    STOP: ofType<string>(),
    TOGGLE: ofType<{ id: string, working: boolean }>()
});

export type ProgressIndicatorAction = UnionOf<typeof progressIndicatorActions>;
