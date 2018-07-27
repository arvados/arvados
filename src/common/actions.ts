// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const goToProject = (uuid: string) => {
    return `/projects/${uuid}`;
};

export const goToCollection = (uuid: string) => {
    return `/collections/${uuid}`;
};