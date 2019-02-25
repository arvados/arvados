// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from '~/common/unionize';
import { Config } from '~/common/config';

export const configActions = unionize({
    CONFIG: ofType<{ config: Config }>(),
});

export type ConfigAction = UnionOf<typeof configActions>;
