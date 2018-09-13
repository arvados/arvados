// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export const progressIndicatorActions = unionize({
    START_SUBMIT: ofType<{ id: string }>(),
    STOP_SUBMIT: ofType<{ id: string }>()
});

export type ProgressIndicatorAction = UnionOf<typeof progressIndicatorActions>;