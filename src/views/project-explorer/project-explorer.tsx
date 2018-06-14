// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import DataExplorer, { DataExplorerProps } from "../../components/data-explorer/data-explorer";
import { RouteComponentProps } from 'react-router';
import { Project } from '../../models/project';
import { ProjectState, findTreeItem } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { push } from 'react-router-redux';
import projectActions from "../../store/project/project-action";
import { Typography } from '@material-ui/core';
import { Column } from '../../components/data-explorer/column';
import ProjectExplorer from '../../components/project-explorer/project-explorer';

interface ProjectExplorerViewDataProps {
    projects: ProjectState;
}

type ProjectExplorerViewProps = ProjectExplorerViewDataProps & RouteComponentProps<{ name: string }> & DispatchProp;

type ProjectExplorerViewState = Pick<DataExplorerProps<Project>, "columns">;

class ProjectExplorerView extends React.Component<ProjectExplorerViewProps, ProjectExplorerViewState> {

    render() {
        const project = findTreeItem(this.props.projects, this.props.match.params.name);
        const projectItems = project && project.items || [];
        return (
            <ProjectExplorer
                projects={projectItems.map(item => item.data)}
                onProjectClick={this.goToProject}
            />
        );
    }

    goToProject = (project: Project) => {
        this.props.dispatch(push(`/project/${project.uuid}`));
        this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(project.uuid));
    }

}

export default connect(
    (state: RootState) => ({
        projects: state.projects
    })
)(ProjectExplorerView);
