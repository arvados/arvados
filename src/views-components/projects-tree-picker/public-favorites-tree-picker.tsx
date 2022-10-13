// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { ProjectsTreePicker, ProjectsTreePickerProps } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { Dispatch } from 'redux';
import { PublicFavoriteIcon } from 'components/icon/icon';
import { loadPublicFavoritesProject } from 'store/tree-picker/tree-picker-actions';

export const PublicFavoritesTreePicker = connect(() => ({
    rootItemIcon: PublicFavoriteIcon,
}), (dispatch: Dispatch): Pick<ProjectsTreePickerProps, 'loadRootItem'> => ({
    loadRootItem: (_, pickerId, includeCollections, includeFiles, options) => {
        dispatch<any>(loadPublicFavoritesProject({ pickerId, includeCollections, includeFiles, options }));
    },
}))(ProjectsTreePicker);