// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { configActions, ConfigAction } from "./config-action";
import { mockConfig } from '~/common/config';

export const configReducer = (state = mockConfig({}), action: ConfigAction) => {
    return configActions.match(action, {
        CONFIG: ({ config }) => {
            return {
                ...state, ...config
            };
        },
        default: () => state
    });
};
