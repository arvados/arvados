// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ownerNameUuidEnhancerActions, OwnerNameUuidEnhancerAction, OwnerNameState } from './owner-name-uuid-enhancer-actions';

export const ownerNameUuidEnhancerReducer = (state = {}, action: OwnerNameUuidEnhancerAction) =>
    ownerNameUuidEnhancerActions.match(action, {
        SET_OWNER_NAME_BY_UUID: (data: OwnerNameState) => ({...state, [data.uuid]: data.name }),
        default: () => state,
    });