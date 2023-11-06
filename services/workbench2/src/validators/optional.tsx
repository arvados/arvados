// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const optional = (validator: (value: any) => string | undefined) =>
    (value: any) =>
        value === undefined || value === null || value === ''  ? undefined : validator(value);