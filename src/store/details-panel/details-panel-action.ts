// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { CommonResourceService } from "../../common/api/common-resource-service";
import { Dispatch } from "redux";
import { serverApi } from "../../common/api/server-api";
import { Resource, ResourceKind } from "../../models/resource";

export const detailsPanelActions = unionize({
    TOGGLE_DETAILS_PANEL: ofType<{}>(),
    LOAD_DETAILS: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_DETAILS_SUCCESS: ofType<{ item: Resource }>(),
}, { tag: 'type', value: 'payload' });

export type DetailsPanelAction = UnionOf<typeof detailsPanelActions>;

export const loadDetails = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch) => {
        dispatch(detailsPanelActions.LOAD_DETAILS({ uuid, kind }));
        getService(kind)
            .get(uuid)
            .then(project => {
                dispatch(detailsPanelActions.LOAD_DETAILS_SUCCESS({ item: project }));
            });
    };

const getService = (kind: ResourceKind) => {
    switch (kind) {
        case ResourceKind.Project:
            return new CommonResourceService(serverApi, "groups");
        case ResourceKind.Collection:
            return new CommonResourceService(serverApi, "collections");
        default:
            return new CommonResourceService(serverApi, "");
    }
};



