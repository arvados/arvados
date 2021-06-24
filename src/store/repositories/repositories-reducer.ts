// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { repositoriesActions, RepositoriesActions } from 'store/repositories/repositories-actions';
import { RepositoryResource } from 'models/repositories';

interface Repositories {
    items: RepositoryResource[];
}

const initialState: Repositories = {
    items: []
};

export const repositoriesReducer = (state = initialState, action: RepositoriesActions): Repositories =>
    repositoriesActions.match(action, {
        SET_REPOSITORIES: items => ({ ...state, items }),
        default: () => state
    });