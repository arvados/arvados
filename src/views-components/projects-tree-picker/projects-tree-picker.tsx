// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { values, memoize, pipe } from 'lodash/fp';
import { HomeTreePicker } from 'views-components/projects-tree-picker/home-tree-picker';
import { SharedTreePicker } from 'views-components/projects-tree-picker/shared-tree-picker';
import { FavoritesTreePicker } from 'views-components/projects-tree-picker/favorites-tree-picker';
import { getProjectsTreePickerIds, SHARED_PROJECT_ID, FAVORITES_PROJECT_ID } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from './generic-projects-tree-picker';
import { PublicFavoritesTreePicker } from './public-favorites-tree-picker';

export interface ProjectsTreePickerProps {
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
    showSelection?: boolean;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    toggleItemActive?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectsTreePickerItem>, pickerId: string) => void;
    toggleItemSelection?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectsTreePickerItem>, pickerId: string) => void;
}

export const ProjectsTreePicker = ({ pickerId, ...props }: ProjectsTreePickerProps) => {
    const { home, shared, favorites, publicFavorites } = getProjectsTreePickerIds(pickerId);
    const relatedTreePickers = getRelatedTreePickers(pickerId);
    const p = {
        ...props,
        relatedTreePickers,
        disableActivation
    };
    return <div>
        <HomeTreePicker pickerId={home} {...p} />
        <SharedTreePicker pickerId={shared} {...p} />
        <PublicFavoritesTreePicker pickerId={publicFavorites} {...p} />
        <div data-cy="projects-tree-favourites-tree-picker">
            <FavoritesTreePicker pickerId={favorites} {...p} />  
        </div>
    </div>;
};

const getRelatedTreePickers = memoize(pipe(getProjectsTreePickerIds, values));
const disableActivation = [SHARED_PROJECT_ID, FAVORITES_PROJECT_ID];
