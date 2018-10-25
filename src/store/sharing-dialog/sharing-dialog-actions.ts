// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "~/store/dialog/dialog-actions";
import { withDialog } from "~/store/dialog/with-dialog";

export const SHARING_DIALOG_NAME = 'SHARING_DIALOG_NAME';

export const openSharingDialog = (resourceUuid: string) =>
    dialogActions.OPEN_DIALOG({ id: SHARING_DIALOG_NAME, data: resourceUuid });

export const closeSharingDialog = () =>
    dialogActions.CLOSE_DIALOG({ id: SHARING_DIALOG_NAME });

export const connectSharingDialog = withDialog(SHARING_DIALOG_NAME);
