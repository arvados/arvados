// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


const ERROR_MESSAGE = 'Public key is invalid';

export const isRsaKey = (value: any) => {
    return value.match(/ssh-rsa AAAA[0-9A-Za-z+/]+[=]{0,3}(( [^@]+@[^@]+)|$)/i) ? undefined : ERROR_MESSAGE;
};
