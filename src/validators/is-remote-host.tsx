// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


const ERROR_MESSAGE = 'Remote host is invalid';

export const isRemoteHost = (value: string) => {
    return value.match(/\w+\.\w+\.\w+/i) ? undefined : ERROR_MESSAGE;
};
