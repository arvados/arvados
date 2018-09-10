// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const PROCESS_COPY_FORM_NAME = 'processCopyFormName';

export interface ProcessCopyFormDialogData {
    name: string;
    ownerUuid: string;
    uuid: string;
}