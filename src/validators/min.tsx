// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isNumber } from 'lodash';

export const ERROR_MESSAGE = (minValue: number) => `Minimum value is ${minValue}`;

export const min =
    (minValue: number, errorMessage = ERROR_MESSAGE) =>
        (value: any) =>
            isNumber(value) && value >= minValue ? undefined : errorMessage(minValue);
