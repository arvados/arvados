// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const disallowDotName = /^\.{1,2}$/;
export const disallowSlash = /\//;

const ERROR_MESSAGE = "Name cannot be '.' or '..' or contain '/' characters";

export const validName = (value: string) => {
    return [disallowDotName, disallowSlash].find(aRule => value.match(aRule) !== null)
        ? ERROR_MESSAGE
        : undefined;
};

export const validNameAllowSlash = (value: string) => {
    return [disallowDotName].find(aRule => value.match(aRule) !== null)
        ? "Name cannot be '.' or '..'"
        : undefined;
};
