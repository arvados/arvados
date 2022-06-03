// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { propertiesActions } from "store/properties/properties-actions";
import { BREADCRUMBS } from "./breadcrumbs-actions";

export const breadcrumbsMiddleware = store => next => action => {
    propertiesActions.match(action, {
        SET_PROPERTY: () => {

            if (action.payload.key === BREADCRUMBS && Array.isArray(action.payload.value)) {
                action.payload.value = action.payload
                    .value.map((value)=> ({ ...value, isFrozen: !!store.getState().resources[value.uuid]?.frozenByUuid }));
            }

            next(action);
        },
        default: () => next(action)
    });
};
