// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Vocabulary {
    strict: boolean;
    tags: Tag[];
}

export interface Tag {
    strict: boolean;
    values: string[];
}
