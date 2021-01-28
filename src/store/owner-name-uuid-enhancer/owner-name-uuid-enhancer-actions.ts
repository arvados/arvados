// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { extractUuidObjectType, ResourceObjectType } from '~/models/resource';
import { ServiceRepository } from '~/services/services';
import { RootState } from '../store';

export type OwnerNameUuidEnhancerAction = UnionOf<typeof ownerNameUuidEnhancerActions>;

export interface OwnerNameState {
    name: string;
    uuid: string;
}

export const ownerNameUuidEnhancerActions = unionize({
    SET_OWNER_NAME_BY_UUID: ofType<OwnerNameState>()
});

export const fetchOwnerNameByUuid = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const objectType = extractUuidObjectType(uuid);

        switch (objectType) {
            case ResourceObjectType.USER:
                services.userService.get(uuid, false)
                    .then((data) =>
                        dispatch(
                            ownerNameUuidEnhancerActions.SET_OWNER_NAME_BY_UUID({
                                uuid,
                                name: (data as any).fullName,
                            })
                        )
                    );
                break;
            case ResourceObjectType.GROUP:
                services.groupsService.get(uuid, false)
                    .then((data) =>
                        dispatch(
                            ownerNameUuidEnhancerActions.SET_OWNER_NAME_BY_UUID({
                                uuid,
                                name: (data as any).name,
                            })
                        )
                    );
                break;
            default:
                break;
        }
    };