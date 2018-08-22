// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { Resource } from '~/models/resource';

export const resourcesActions = unionize({
    SET_RESOURCES: ofType<Resource[]>(),
    DELETE_RESOURCES: ofType<string[]>()
});

export type ResourcesAction = UnionOf<typeof resourcesActions>;