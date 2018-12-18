// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const ERROR_MESSAGE = (minLength: number) => `Min length is ${minLength}`;

export const minLength =
    (minLength: number, errorMessage = ERROR_MESSAGE) =>
        (value: { length: number }) =>
            value && value.length >= minLength ? undefined : errorMessage(minLength);
