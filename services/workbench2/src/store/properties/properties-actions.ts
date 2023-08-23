// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from 'common/unionize';

export const propertiesActions = unionize({
    SET_PROPERTY: ofType<{ key: string, value: any }>(),
    DELETE_PROPERTY: ofType<string>(),
});

export type PropertiesAction = UnionOf<typeof propertiesActions>;
