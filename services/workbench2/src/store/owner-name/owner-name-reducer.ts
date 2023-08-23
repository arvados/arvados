// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ownerNameActions, OwnerNameAction } from './owner-name-actions';

export const ownerNameReducer = (state = [], action: OwnerNameAction) =>
    ownerNameActions.match(action, {
        SET_OWNER_NAME: data => [...state, { uuid: data.uuid, name: data.name }],
        default: () => state,
    });