// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const ERROR_MESSAGE = 'This field is required.';

interface RequireProps {
    value: string;
}

// TODO types for require
const require: any = (value: string, errorMessage = ERROR_MESSAGE) => {
    return value && value.toString().length > 0 ? undefined : ERROR_MESSAGE;
};

export default require;
