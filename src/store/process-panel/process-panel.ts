// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RouterState } from "react-router-redux";
import { matchProcessRoute } from "routes/routes";

export interface ProcessPanel {
    containerRequestUuid: string;
    filters: { [status: string]: boolean };
}

export const getProcessPanelCurrentUuid = (router: RouterState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    return match ? match.params.id : undefined;
};