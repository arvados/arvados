// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { push } from "react-router-redux";
import { ResourceKind, extractUuidKind } from '~/models/resource';
import { getCollectionUrl } from "~/models/collection";
import { getProjectUrl } from "~/models/project";

import { SidePanelTreeCategory } from '../side-panel-tree/side-panel-tree-actions';
import { Routes, getProcessUrl } from '~/routes/routes';

export const navigateTo = (uuid: string) =>
    async (dispatch: Dispatch) => {
        const kind = extractUuidKind(uuid);
        if (kind === ResourceKind.PROJECT || kind === ResourceKind.USER) {
            dispatch<any>(navigateToProject(uuid));
        } else if (kind === ResourceKind.COLLECTION) {
            dispatch<any>(navigateToCollection(uuid));
        } else if (kind === ResourceKind.CONTAINER_REQUEST) {
            dispatch<any>(navigateToProcess(uuid));
        }
        if (uuid === SidePanelTreeCategory.FAVORITES) {
            dispatch<any>(navigateToFavorites);
        }
    };

export const navigateToFavorites = push(Routes.FAVORITES);

export const navigateToTrash = push(Routes.TRASH);

export const navigateToProject = compose(push, getProjectUrl);

export const navigateToCollection = compose(push, getCollectionUrl);

export const navigateToProcess = compose(push, getProcessUrl);
