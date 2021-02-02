// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { extractUuidKind, Resource } from '~/models/resource';
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { getResourceService } from '~/services/services';

export const resourcesActions = unionize({
    SET_RESOURCES: ofType<Resource[]>(),
    DELETE_RESOURCES: ofType<string[]>()
});

export type ResourcesAction = UnionOf<typeof resourcesActions>;

export const updateResources = (resources: Resource[]) => resourcesActions.SET_RESOURCES(resources);

export const loadResource = (uuid: string, showErrors?: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const kind = extractUuidKind(uuid);
            const service = getResourceService(kind)(services);
            if (service) {
                const resource = await service.get(uuid, showErrors);
                dispatch<any>(updateResources([resource]));
                return resource;
            }
        } catch {}
        return undefined;
    };
