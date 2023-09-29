// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from 'common/unionize';

export const ownerNameActions = unionize({
    SET_OWNER_NAME: ofType<OwnerNameState>()
});

interface OwnerNameState {
    name: string;
    uuid: string;
}

export type OwnerNameAction = UnionOf<typeof ownerNameActions>;
