// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


const ERROR_MESSAGE = "Name cannot be '.' or '..' or contain '/' characters";

export const invalidNamingRules = [/\//, /^\.{1,2}$/];

export const validName = (value: string) => {
    return invalidNamingRules.find(aRule => value.match(aRule) !== null)
        ? ERROR_MESSAGE
        : undefined;
};
