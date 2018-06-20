// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RouteComponentProps } from 'react-router';
import { Project } from '../../models/project';
import { ProjectState, findTreeItem } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { push } from 'react-router-redux';
import projectActions from "../../store/project/project-action";
import { DataColumns } from "../../components/data-table/data-table";
import DataExplorer, { DataExplorerContextActions } from "../../views-components/data-explorer/data-explorer";
import { mapProjectTreeItem } from "./data-explorer-selectors";
import { DataItem } from "../../views-components/data-explorer/data-item";

interface DataExplorerViewDataProps {
    projects: ProjectState;
}

type DataExplorerViewProps = DataExplorerViewDataProps & RouteComponentProps<{ name: string }> & DispatchProp;
type DataExplorerViewState = DataColumns<Project>;

class DataExplorerView extends React.Component<DataExplorerViewProps, DataExplorerViewState> {

    render() {
        const project = findTreeItem(this.props.projects, this.props.match.params.name);
        const projectItems = project && project.items || [];
        return (
            <DataExplorer
                items={projectItems.map(mapProjectTreeItem)}
                onItemClick={this.goToProject}
                contextActions={this.contextActions}
            />
        );
    }

    contextActions: DataExplorerContextActions = {
        onAddToFavourite: console.log,
        onCopy: console.log,
        onDownload: console.log,
        onMoveTo: console.log,
        onRemove: console.log,
        onRename: console.log,
        onShare: console.log
    };

    goToProject = (item: DataItem) => {
        this.props.dispatch(push(`/project/${item}`));
        this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(item.uuid));
    }
}

export default connect(
    (state: RootState) => ({
        projects: state.projects
    })
)(DataExplorerView);
