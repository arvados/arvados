// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type PublicFavoritesState = Record<string, boolean>;

export const checkPublicFavorite = (uuid: string, state: PublicFavoritesState) => state[uuid] === true;
