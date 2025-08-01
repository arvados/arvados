// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


const ERROR_MESSAGE = 'Filename must end in .zip';

export const isZipFilename = (value: any) => {
    return value.match(/\.zip$/i) ? undefined : ERROR_MESSAGE;
};
