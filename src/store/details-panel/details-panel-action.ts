// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';

export const detailsPanelActions = unionize({
    TOGGLE_DETAILS_PANEL: ofType<{}>(),
    LOAD_DETAILS_PANEL: ofType<string>()
});

export type DetailsPanelAction = UnionOf<typeof detailsPanelActions>;

export const loadDetailsPanel = (uuid: string) => detailsPanelActions.LOAD_DETAILS_PANEL(uuid);




