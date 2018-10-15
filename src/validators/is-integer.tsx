// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isInteger as isInt } from 'lodash';

const ERROR_MESSAGE = 'This field can only contain integer values';

export const isInteger = (value: any) => {
    return isInt(value) ? undefined : ERROR_MESSAGE;
};
