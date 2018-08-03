// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type KeepManifest = KeepManifestStream[];

export interface KeepManifestStream {
    name: string;
    locators: string[];
    files: Array<KeepManifestStreamFile>;
}

export interface KeepManifestStreamFile {
    name: string;
    position: string;
    size: number;
}
