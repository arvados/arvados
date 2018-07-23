// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";

export const favoritesActions = unionize({
    UPDATE_FAVORITES: ofType<Record<string, boolean>>()
}, { tag: 'type', value: 'payload' });

export type FavoritesAction = UnionOf<typeof favoritesActions>;