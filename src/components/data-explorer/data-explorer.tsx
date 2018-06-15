// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataTable, DataTableProps, Column, ColumnsConfigurator } from "../../components/data-table";
import { Typography, Grid, ListItem, Divider, List, ListItemIcon, ListItemText } from '@material-ui/core';
import IconButton, { IconButtonProps } from '@material-ui/core/IconButton';
import MoreVertIcon from "@material-ui/icons/MoreVert";
import Popover from '../popover/popover';
import { formatFileSize, formatDate } from './formatters';
import { DataItem } from './data-item';

interface DataExplorerProps {
    items: DataItem[];
    onItemClick: (item: DataItem) => void;
}

type DataExplorerState = Pick<DataTableProps<DataItem>, "columns">;

class DataExplorer extends React.Component<DataExplorerProps, DataExplorerState> {

    state: DataExplorerState = {
        columns: [
            {
                header: "Name",
                selected: true,
                render: item => this.renderName(item)
            },
            {
                header: "Status",
                selected: true,
                render: item => renderStatus(item.status)
            },
            {
                header: "Type",
                selected: true,
                render: item => renderType(item.type)
            },
            {
                header: "Owner",
                selected: true,
                render: item => renderOwner(item.owner)
            },
            {
                header: "File size",
                selected: true,
                render: (item) => renderFileSize(item.fileSize)
            },
            {
                header: "Last modified",
                selected: true,
                render: item => renderDate(item.lastModified)
            },
            {
                header: "Actions",
                key: "Actions",
                selected: true,
                configurable: false,
                renderHeader: () => this.renderActionsHeader(),
                render: renderItemActions
            }
        ]
    };

    render() {
        return (
            <DataTable
                columns={this.state.columns}
                items={this.props.items}
            />
        );
    }

    toggleColumn = (column: Column<DataItem>) => {
        const index = this.state.columns.indexOf(column);
        const columns = this.state.columns.slice(0);
        columns.splice(index, 1, { ...column, selected: !column.selected });
        this.setState({ columns });
    }

    renderActionsHeader = () => {
        return (
            <Grid container justify="flex-end">
                <ColumnsConfigurator
                    columns={this.state.columns}
                    onColumnToggle={this.toggleColumn}
                />
            </Grid>
        );
    }

    renderName = (item: DataItem) => {
        return (
            (
                <Grid
                    container
                    alignItems="center"
                    wrap="nowrap"
                    spacing={16}
                    onClick={() => this.props.onItemClick(item)}
                >
                    <Grid item>
                        {renderIcon(item)}
                    </Grid>
                    <Grid item>
                        <Typography color="primary">
                            {item.name}
                        </Typography>
                    </Grid>
                </Grid>
            )
        );
    }
}

const renderIcon = (dataItem: DataItem) => {
    switch (dataItem.type) {
        case "arvados#group":
            return <i className="fas fa-folder fa-lg" />;
        case "arvados#groupList":
            return <i className="fas fa-th fa-lg" />;
        default:
            return <i />;
    }
};

const renderDate = (date: string) => {
    return (
        <Typography noWrap>
            {formatDate(date)}
        </Typography>
    );
};

const renderFileSize = (fileSize?: number) => {
    return (
        <Typography noWrap>
            {typeof fileSize === "number" ? formatFileSize(fileSize) : "-"}
        </Typography>
    );
};

const renderOwner = (owner: string) => {
    return (
        <Typography noWrap color="primary">
            {owner}
        </Typography>
    );
};

const renderType = (type: string) => {
    return (
        <Typography noWrap>
            {type}
        </Typography>
    );
};

const renderStatus = (status?: string) => {
    return (
        <Typography noWrap align="center">
            {status || "-"}
        </Typography>
    );
};

const renderItemActions = () => {
    return (
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
    );
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

export default DataExplorer;
