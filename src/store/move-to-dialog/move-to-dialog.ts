// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface MoveToFormDialogData {
    name: string;
    uuid: string;
    ownerUuid: string;
    isSingle?: boolean;
}
