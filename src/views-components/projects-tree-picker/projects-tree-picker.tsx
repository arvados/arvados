// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { HomeTreePicker } from '~/views-components/projects-tree-picker/home-tree-picker';
import { SharedTreePicker } from '~/views-components/projects-tree-picker/shared-tree-picker';
import { FavoritesTreePicker } from '~/views-components/projects-tree-picker/favorites-tree-picker';
import { getProjectsTreePickerIds, treePickerActions } from '~/store/tree-picker/tree-picker-actions';
import { TreeItem } from '~/components/tree/tree';
import { ProjectsTreePickerItem } from './generic-projects-tree-picker';

export interface ProjectsTreePickerProps {
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
    rootItemSelection?: boolean;
    projectsSelection?: boolean;
    collectionsSelection?: boolean;
    filesSelection?: boolean;
    toggleItemActive?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<ProjectsTreePickerItem>, pickerId: string) => void;
}

export const ProjectsTreePicker = ({ pickerId, ...props }: ProjectsTreePickerProps) => {
    const { home, shared, favorites } = getProjectsTreePickerIds(pickerId);
    return <div>
        <HomeTreePicker pickerId={home} {...props} />
        <SharedTreePicker pickerId={shared} {...props} />
        <FavoritesTreePicker pickerId={favorites} {...props} />
    </div>;
};
