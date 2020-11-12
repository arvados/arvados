// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const disallowDotName = /^\.{1,2}$/;
export const disallowSlash = /\//;
export const disallowLeadingWhitespaces = /^\s+/;
export const disallowTrailingWhitespaces = /\s+$/;

export const validName = (value: string) => {
    return [disallowDotName, disallowSlash].find(aRule => value.match(aRule) !== null)
        ? "Name cannot be '.' or '..' or contain '/' characters"
        : undefined;
};

export const validNameAllowSlash = (value: string) => {
    return [disallowDotName].find(aRule => value.match(aRule) !== null)
        ? "Name cannot be '.' or '..'"
        : undefined;
};

export const validFileName = (value: string) => {
    return [
        disallowLeadingWhitespaces,
        disallowTrailingWhitespaces
    ].find(aRule => value.match(aRule) !== null)
        ? `Leading/trailing whitespaces not allowed on '${value}'`
        : undefined;
};

export const validFilePath = (filePath: string) => {
    const errors = filePath.split('/').map(pathPart => {
        if (pathPart === "") { return "Empty dir name not allowed"; }
        return validNameAllowSlash(pathPart) || validFileName(pathPart);
    });
    return errors.filter(e => e !== undefined)[0];
};