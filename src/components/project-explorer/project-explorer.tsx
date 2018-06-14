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

interface ProjectExplorerProps {
    projects: Project[];
    onProjectClick: (project: Project) => void;
}

type ProjectExplorerState = Pick<DataExplorerProps<Project>, "columns">;

class ProjectExplorer extends React.Component<ProjectExplorerProps, ProjectExplorerState> {

    state: ProjectExplorerState = {
        columns: [
            { header: "Name", selected: true, render: item => <Typography noWrap>{renderIcon(item.kind)} {item.name}</Typography> },
            { header: "Created at", selected: true, render: item => <Typography noWrap>{formatDate(item.createdAt)}</Typography> },
            { header: "Modified at", selected: true, render: item => <Typography noWrap>{formatDate(item.modifiedAt)}</Typography> },
            { header: "UUID", selected: true, render: item => <Typography noWrap>{item.uuid}</Typography> },
            { header: "Owner UUID", selected: true, render: item => <Typography noWrap>{item.ownerUuid}</Typography> },
            { header: "URL", selected: true, render: item => <Typography noWrap>{item.href}</Typography> }
        ]
    };

    render() {
        return (
            <DataExplorer
                columns={this.state.columns}
                items={this.props.projects}
                onItemClick={this.props.onProjectClick}
                onColumnToggle={this.toggleColumn}
            />
        );
    }

    toggleColumn = (column: Column<Project>) => {
        const index = this.state.columns.indexOf(column);
        const columns = this.state.columns.slice(0);
        columns.splice(index, 1, { ...column, selected: !column.selected });
        this.setState({ columns });
    }
}

const formatDate = (isoDate: string) => {
    const date = new Date(isoDate);
    return date.toLocaleString();
};

const renderIcon = (kind: string) => {
    switch (kind) {
        case "arvados#group":
            return <i className="fas fa-folder" />;
        case "arvados#groupList":
            return <i className="fas fa-th" />;
        default:
            return <i />;
    }
};

export default ProjectExplorer;
