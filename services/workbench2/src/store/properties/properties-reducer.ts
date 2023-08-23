// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PropertiesState, setProperty, deleteProperty } from './properties';
import { PropertiesAction, propertiesActions } from './properties-actions';


export const propertiesReducer = (state: PropertiesState = {}, action: PropertiesAction) =>
    propertiesActions.match(action, {
        SET_PROPERTY: ({ key, value }) => setProperty(key, value)(state),
        DELETE_PROPERTY: key => deleteProperty(key)(state),
        default: () => state,
    });