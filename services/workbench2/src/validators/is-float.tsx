// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isNumber } from 'lodash';

const ERROR_MESSAGE = 'This field must be a float';

export const isFloat = (value: any) => {
    return isNumber(value) ? undefined : ERROR_MESSAGE;
};
