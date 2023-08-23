// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isNumber as isNum } from 'lodash';
const ERROR_MESSAGE = 'This field can only contain numeric values';

export const isNumber = (value: any) => {
    return !isNaN(value) && isNum(value) ? undefined : ERROR_MESSAGE;
};
