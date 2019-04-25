// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { initialize } from "redux-form";
import { ServiceRepository } from "~/services/services";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { authActions } from "~/store/auth/auth-action";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";

export const loadLinkAccountPanel = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
       dispatch(setBreadcrumbs([{ label: 'Link account'}]));
    };
