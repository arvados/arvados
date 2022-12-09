// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { ProjectsTreePicker, ProjectsTreePickerProps } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { Dispatch } from 'redux';
import { SearchIcon } from 'components/icon/icon';
import { loadProject } from 'store/tree-picker/tree-picker-actions';
import { SEARCH_PROJECT_ID } from 'store/tree-picker/tree-picker-actions';

export const SearchProjectsPicker = connect(() => ({
    rootItemIcon: SearchIcon,
}), (dispatch: Dispatch): Pick<ProjectsTreePickerProps, 'loadRootItem'> => ({
    loadRootItem: (_, pickerId, includeCollections, includeFiles, options) => {
        dispatch<any>(loadProject({ id: SEARCH_PROJECT_ID, pickerId, includeCollections, includeFiles, searchProjects: true, options }));
    },
}))(ProjectsTreePicker);
