// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const ERROR_MESSAGE = 'Maximum string length of this field is: ';
export const DEFAULT_MAX_VALUE = 60;

export const maxLength: any = (maxLengthValue = DEFAULT_MAX_VALUE, errorMessage = ERROR_MESSAGE) => {
    return (value: string) => {
        if (value) {
            return  value && value.length <= maxLengthValue ? undefined : `${errorMessage || ERROR_MESSAGE} ${maxLengthValue}`;
        }

        return undefined;
    };
};
