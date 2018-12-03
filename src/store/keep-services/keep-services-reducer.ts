// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { keepServicesActions, KeepServicesActions } from '~/store/keep-services/keep-services-actions';
import { KeepServiceResource } from '~/models/keep-services';

export type KeepSericesState = KeepServiceResource[];

const initialState: KeepSericesState = [];

export const keepServicesReducer = (state: KeepSericesState = initialState, action: KeepServicesActions): KeepSericesState =>
    keepServicesActions.match(action, {
        SET_KEEP_SERVICES: items => items,
        REMOVE_KEEP_SERVICE: (uuid: string) => state.filter((keepService) => keepService.uuid !== uuid),
        default: () => state
    });