// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { ProjectsTreePicker, ProjectsTreePickerProps } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { Dispatch } from 'redux';
import { ShareMeIcon } from 'components/icon/icon';
import { loadProject } from 'store/tree-picker/tree-picker-actions';
import { SHARED_PROJECT_ID } from 'store/tree-picker/tree-picker-actions';

export const SharedTreePicker = connect(() => ({
    rootItemIcon: ShareMeIcon,
}), (dispatch: Dispatch): Pick<ProjectsTreePickerProps, 'loadRootItem'> => ({
    loadRootItem: (_, pickerId, includeCollections, includeFiles, options) => {
        dispatch<any>(loadProject({ id: SHARED_PROJECT_ID, pickerId, includeCollections, includeFiles, loadShared: true, options }));
    },
}))(ProjectsTreePicker);
