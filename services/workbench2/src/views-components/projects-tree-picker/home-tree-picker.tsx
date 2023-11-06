// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { ProjectsTreePicker, ProjectsTreePickerProps } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { Dispatch } from 'redux';
import { loadUserProject } from 'store/tree-picker/tree-picker-actions';
import { ProjectsIcon } from 'components/icon/icon';

export const HomeTreePicker = connect(() => ({
    rootItemIcon: ProjectsIcon,
}), (dispatch: Dispatch): Pick<ProjectsTreePickerProps, 'loadRootItem'> => ({
    loadRootItem: (_, pickerId, includeCollections, includeDirectories, includeFiles, options) => {
        dispatch<any>(loadUserProject(pickerId, includeCollections, includeDirectories, includeFiles, options));
    },
}))(ProjectsTreePicker);
