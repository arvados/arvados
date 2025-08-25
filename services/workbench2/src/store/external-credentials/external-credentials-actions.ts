// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";

export const EXTERNAL_CREDENTIALS_PANEL = 'externalCredentialsPanel';

export const externalCredentialsActions = unionize({
});

export type ExternalCredentialsAction = UnionOf<typeof externalCredentialsActions>;

