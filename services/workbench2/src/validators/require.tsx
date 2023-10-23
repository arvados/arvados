// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const ERROR_MESSAGE = 'This field is required.';

export const require: any = (value: string) => {
    return value && value.length > 0 ? undefined : ERROR_MESSAGE;
};
