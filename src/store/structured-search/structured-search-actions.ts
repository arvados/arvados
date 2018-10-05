// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

export const structuredSearchActions = unionize({
    SET_CURRENT_VIEW: ofType<string>()
});

export type StructuredSearchActions = UnionOf<typeof structuredSearchActions>;

export const goToView = (currentView: string) => structuredSearchActions.SET_CURRENT_VIEW(currentView);