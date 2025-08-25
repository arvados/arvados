// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ExternalCredentialsAction, externalCredentialsActions } from "./external-credentials-actions";

export type ExternalCredentialsState = Record<string, boolean>;

export const externalCredentialsReducer = (state: ExternalCredentialsState = {}, action: ExternalCredentialsAction) =>
    externalCredentialsActions.match(action, {
        default: () => state
    });