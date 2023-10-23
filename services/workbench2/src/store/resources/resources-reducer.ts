// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourcesState, setResource, deleteResource } from './resources';
import { ResourcesAction, resourcesActions } from './resources-actions';

export const resourcesReducer = (state: ResourcesState = {}, action: ResourcesAction) =>
    resourcesActions.match(action, {
        SET_RESOURCES: resources => resources.reduce(
            (state, resource) => setResource(resource.uuid, resource)(state),
            state),
        DELETE_RESOURCES: ids => ids.reduce(
            (state, id) => deleteResource(id)(state),
            state),
        default: () => state,
    });