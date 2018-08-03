// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { Resource, ResourceKind } from "../../models/resource";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";

export const detailsPanelActions = unionize({
    TOGGLE_DETAILS_PANEL: ofType<{}>(),
    LOAD_DETAILS: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_DETAILS_SUCCESS: ofType<{ item: Resource }>(),
}, { tag: 'type', value: 'payload' });

export type DetailsPanelAction = UnionOf<typeof detailsPanelActions>;

export const loadDetails = (uuid: string, kind: ResourceKind) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(detailsPanelActions.LOAD_DETAILS({ uuid, kind }));
        const item = await getService(services, kind).get(uuid);
        dispatch(detailsPanelActions.LOAD_DETAILS_SUCCESS({ item }));
    };

const getService = (services: ServiceRepository, kind: ResourceKind) => {
    switch (kind) {
        case ResourceKind.PROJECT:
            return services.projectService;
        case ResourceKind.COLLECTION:
            return services.collectionService;
        default:
            return services.projectService;
    }
};



