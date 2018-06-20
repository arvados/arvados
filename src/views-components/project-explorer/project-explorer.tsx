// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataExplorerColumn } from '../../components/data-explorer/data-explorer-column';
import { ProjectExplorerItem } from './project-explorer-item';
import { Grid, Typography } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import DataExplorer from '../../components/data-explorer/data-explorer';

export interface ProjectExplorerContextActions {
    onAddToFavourite: (item: ProjectExplorerItem) => void;
    onCopy: (item: ProjectExplorerItem) => void;
    onDownload: (item: ProjectExplorerItem) => void;
    onMoveTo: (item: ProjectExplorerItem) => void;
    onRemove: (item: ProjectExplorerItem) => void;
    onRename: (item: ProjectExplorerItem) => void;
    onShare: (item: ProjectExplorerItem) => void;
}

interface ProjectExplorerProps {
    items: ProjectExplorerItem[];
}

interface ProjectExplorerState {
    columns: Array<DataExplorerColumn<ProjectExplorerItem>>;
}

class ProjectExplorer extends React.Component<ProjectExplorerProps, ProjectExplorerState> {
    state: ProjectExplorerState = {
        columns: [{
            name: "Name",
            selected: true,
            sortDirection: "asc",
            sortable: true,
            render: renderName
        }, {
            name: "Status",
            selected: true,
            filterable: true,
            filters: [{
                name: "In progress",
                selected: true
            }, {
                name: "Complete",
                selected: true
            }],
            render: renderStatus
        }, {
            name: "Type",
            selected: true,
            filterable: true,
            filters: [{
                name: "Collection",
                selected: true
            }, {
                name: "Group",
                selected: true
            }],
            render: item => renderType(item.type)
        }, {
            name: "Owner",
            selected: true,
            render: item => renderOwner(item.owner)
        }, {
            name: "File size",
            selected: true,
            render: item => renderFileSize(item.fileSize)
        }, {
            name: "Last modified",
            selected: true,
            sortable: true,
            render: item => renderDate(item.lastModified)
        }]
    };

    contextMenuActions = [[{
        icon: "fas fa-users fa-fw",
        name: "Share",
        onClick: console.log
    }, {
        icon: "fas fa-sign-out-alt fa-fw",
        name: "Move to",
        onClick: console.log
    }, {
        icon: "fas fa-star fa-fw",
        name: "Add to favourite",
        onClick: console.log
    }, {
        icon: "fas fa-edit fa-fw",
        name: "Rename",
        onClick: console.log
    }, {
        icon: "fas fa-copy fa-fw",
        name: "Make a copy",
        onClick: console.log
    }, {
        icon: "fas fa-download fa-fw",
        name: "Download",
        onClick: console.log
    }], [{
        icon: "fas fa-trash-alt fa-fw",
        name: "Remove",
        onClick: console.log
    }
    ]];

    render() {
        return <DataExplorer
            items={this.props.items}
            columns={this.state.columns}
            contextActions={this.contextMenuActions}
            onColumnToggle={console.log}
            onFiltersChange={console.log}
            onRowClick={console.log}
            onSortingToggle={console.log} />;
    }
}

const renderName = (item: ProjectExplorerItem) =>
    <Grid
        container
        alignItems="center"
        wrap="nowrap"
        spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary">
                {item.name}
            </Typography>
        </Grid>
    </Grid>;

const renderIcon = (item: ProjectExplorerItem) => {
    switch (item.type) {
        case "arvados#group":
            return <i className="fas fa-folder fa-lg" />;
        case "arvados#groupList":
            return <i className="fas fa-th fa-lg" />;
        default:
            return <i />;
    }
};

const renderDate = (date: string) =>
    <Typography noWrap>
        {formatDate(date)}
    </Typography>;

const renderFileSize = (fileSize?: number) =>
    <Typography noWrap>
        {formatFileSize(fileSize)}
    </Typography>;

const renderOwner = (owner: string) =>
    <Typography noWrap color="primary">
        {owner}
    </Typography>;

const renderType = (type: string) =>
    <Typography noWrap>
        {type}
    </Typography>;

const renderStatus = (item: ProjectExplorerItem) =>
    <Typography noWrap align="center">
        {item.status || "-"}
    </Typography>;

export default ProjectExplorer;
