// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RouteComponentProps } from 'react-router-dom';
import { DispatchProp, connect } from 'react-redux';
import { ProjectState, findTreeItem } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import ProjectExplorer from '../../views-components/project-explorer/project-explorer';

interface ProjectPanelDataProps {
    projects: ProjectState;
}

type ProjectPanelProps = ProjectPanelDataProps & RouteComponentProps<{ name: string }> & DispatchProp;

class ProjectPanel extends React.Component<ProjectPanelProps> {

    render() {
        return (
            <ProjectExplorer />
        );
    }
}

export default connect(
    (state: RootState) => ({
        projects: state.projects
    })
)(ProjectPanel);
