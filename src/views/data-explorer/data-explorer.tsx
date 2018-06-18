// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataTableProps } from "../../components/data-table";
import { RouteComponentProps } from 'react-router';
import { Project } from '../../models/project';
import { ProjectState, findTreeItem } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { push } from 'react-router-redux';
import projectActions from "../../store/project/project-action";
import { DataExplorer, DataItem } from '../../components/data-explorer';
import { TreeItem } from '../../components/tree/tree';

interface DataExplorerViewDataProps {
    projects: ProjectState;
}

type DataExplorerViewProps = DataExplorerViewDataProps & RouteComponentProps<{ name: string }> & DispatchProp;

type DataExplorerViewState = Pick<DataTableProps<Project>, "columns">;

interface MappedProjectItem extends DataItem {
    uuid: string;
}

class DataExplorerView extends React.Component<DataExplorerViewProps, DataExplorerViewState> {

    render() {
        const project = findTreeItem(this.props.projects, this.props.match.params.name);
        const projectItems = project && project.items || [];
        return (
            <DataExplorer
                items={projectItems.map(mapTreeItem)}
                onItemClick={this.goToProject}
            />
        );
    }

    goToProject = (project: MappedProjectItem) => {
        this.props.dispatch(push(`/project/${project.uuid}`));
        this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(project.uuid));
    }

}

const mapTreeItem = (item: TreeItem<Project>): MappedProjectItem => ({
    name: item.data.name,
    type: item.data.kind,
    owner: item.data.ownerUuid,
    lastModified: item.data.modifiedAt,
    uuid: item.data.uuid
});


export default connect(
    (state: RootState) => ({
        projects: state.projects
    })
)(DataExplorerView);
