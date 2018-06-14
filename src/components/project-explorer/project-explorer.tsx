// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import DataExplorer, { DataExplorerProps } from "../../components/data-explorer/data-explorer";
import ColumnsConfigurator from "../../components/data-explorer/columns-configurator/columns-configurator";
import { Typography, Grid, ListItem, Divider, List, ListItemIcon, ListItemText } from '@material-ui/core';
import { Column } from '../../components/data-explorer/column';
import IconButton, { IconButtonProps } from '@material-ui/core/IconButton';
import MoreVertIcon from "@material-ui/icons/MoreVert";
import Popover from '../popover/popover';

export interface ProjectItem {
    name: string;
    type: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}

interface ProjectExplorerProps {
    items: ProjectItem[];
    onItemClick: (item: ProjectItem) => void;
}

type ProjectExplorerState = Pick<DataExplorerProps<ProjectItem>, "columns">;

class ProjectExplorer extends React.Component<ProjectExplorerProps, ProjectExplorerState> {

    state: ProjectExplorerState = {
        columns: [
            {
                header: "Name",
                selected: true,
                render: item => (
                    <Grid container onClick={() => this.props.onItemClick(item)}>
                        {renderIcon(item)}
                        <Typography style={{ marginLeft: 8 }}>
                            {item.name}
                        </Typography>
                    </Grid>
                )
            },
            {
                header: "Status",
                selected: true,
                render: item => (
                    <Typography noWrap align="center">
                        {item.status || "-"}
                    </Typography>
                )
            },
            {
                header: "Type",
                selected: true,
                render: item => (
                    <Typography noWrap>
                        {item.type}
                    </Typography>
                )
            },
            {
                header: "Owner",
                selected: true,
                render: item => (
                    <Typography noWrap>
                        {item.owner}
                    </Typography>
                )
            },
            {
                header: "File size",
                selected: true,
                render: ({ fileSize }) => (
                    <Typography noWrap>
                        {typeof fileSize === "number" ? formatFileSize(fileSize) : "-"}
                    </Typography>
                )
            },
            {
                header: "Last modified",
                selected: true,
                render: item => (
                    <Typography noWrap>
                        {formatDate(item.lastModified)}
                    </Typography>
                )
            },
            {
                header: "Actions",
                key: "Actions",
                selected: true,
                configurable: false,
                renderHeader: () => (
                    <Grid container justify="flex-end">
                        <ColumnsConfigurator
                            columns={this.state.columns}
                            onColumnToggle={this.toggleColumn}
                        />
                    </Grid>
                ),
                render: item => (
                    <Grid container justify="flex-end">
                        <Popover triggerComponent={ItemActionsTrigger}>
                            <List dense>
                                {[
                                    {
                                        icon: "fas fa-users",
                                        label: "Share"
                                    },
                                    {
                                        icon: "fas fa-sign-out-alt",
                                        label: "Move to"
                                    },
                                    {
                                        icon: "fas fa-star",
                                        label: "Add to favourite"
                                    },
                                    {
                                        icon: "fas fa-edit",
                                        label: "Rename"
                                    },
                                    {
                                        icon: "fas fa-copy",
                                        label: "Make a copy"
                                    },
                                    {
                                        icon: "fas fa-download",
                                        label: "Download"
                                    }].map(renderAction)
                                }
                                < Divider />
                                {
                                    renderAction({ icon: "fas fa-trash-alt", label: "Remove" })
                                }
                            </List>
                        </Popover>
                    </Grid>
                )
            }
        ]
    };

    render() {
        return (
            <DataExplorer
                columns={this.state.columns}
                items={this.props.items}
                onColumnToggle={this.toggleColumn}
            />
        );
    }

    toggleColumn = (column: Column<ProjectItem>) => {
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

const formatFileSize = (size: number) => {
    switch (true) {
        case size > 1000000000000:
            return `${size / 1000000000000} TB`;
        case size > 1000000000:
            return `${size / 1000000000} GB`;
        case size > 1000000:
            return `${size / 1000000} MB`;
        case size > 1000:
            return `${size / 1000} KB`;
        default:
            return `${size} B`;
    }
};

const renderIcon = (projectItem: ProjectItem) => {
    switch (projectItem.type) {
        case "arvados#group":
            return <i className="fas fa-folder" />;
        case "arvados#groupList":
            return <i className="fas fa-th" />;
        default:
            return <i />;
    }
};

const renderAction = (action: { label: string, icon: string }, index?: number) => (
    <ListItem button key={index}>
        <ListItemIcon>
            <i className={action.icon} />
        </ListItemIcon>
        <ListItemText>
            {action.label}
        </ListItemText>
    </ListItem>
);

const ItemActionsTrigger: React.SFC<IconButtonProps> = (props) => (
    <IconButton {...props}>
        <MoreVertIcon />
    </IconButton>
);

export default ProjectExplorer;
