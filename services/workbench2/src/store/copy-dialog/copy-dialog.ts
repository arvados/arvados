// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface CopyFormDialogData {
    name: string;
    uuid: string;
    ownerUuid: string;
    fromContextMenu?: boolean;
}
