// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ActionType, createStandardAction } from "typesafe-actions";

const actions = {
    saveApiToken: createStandardAction('@@auth/saveApiToken')<string>(),
    getUserTokenDetails: createStandardAction('@@auth/userTokenDetails')(),
    login: createStandardAction('@@auth/login')(),
    logout: createStandardAction('@@auth/logout')()
};

export type AuthAction = ActionType<typeof actions>;
export default actions;
