// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataTable, DataColumn, ColumnSelector } from "../../components/data-table";
import { Typography, Grid, Paper, Toolbar } from '@material-ui/core';
import IconButton from '@material-ui/core/IconButton';
import MoreVertIcon from "@material-ui/icons/MoreVert";
import { formatFileSize, formatDate } from '../../common/formatters';
import { DataItem } from './data-item';
import { mockAnchorFromMouseEvent } from '../popover/helpers';
import { ContextMenu } from './context-menu';

interface DataExplorerProps {
    items: DataItem[];
    onItemClick: (item: DataItem) => void;
}

interface DataExplorerState {
    columns: Array<DataColumn<DataItem>>;
    contextMenu: {
        anchorEl?: HTMLElement;
        item?: DataItem;
    };
}

class DataExplorer extends React.Component<DataExplorerProps, DataExplorerState> {
    state: DataExplorerState = {
        contextMenu: {},
        columns: [
            {
                name: "Name",
                selected: true,
                render: item => this.renderName(item)
            },
            {
                name: "Status",
                selected: true,
                render: item => renderStatus(item.status)
            },
            {
                name: "Type",
                selected: true,
                render: item => renderType(item.type)
            },
            {
                name: "Owner",
                selected: true,
                render: item => renderOwner(item.owner)
            },
            {
                name: "File size",
                selected: true,
                render: item => renderFileSize(item.fileSize)
            },
            {
                name: "Last modified",
                selected: true,
                render: item => renderDate(item.lastModified)
            },
            {
                name: "Actions",
                selected: true,
                configurable: false,
                renderHeader: () => null,
                render: item => this.renderActions(item)
            }
        ]
    };

    render() {
        return <Paper>
            <ContextMenu {...this.state.contextMenu} onClose={this.closeContextMenu} />
            <Toolbar>
                <Grid container justify="flex-end">
                    <ColumnSelector
                        columns={this.state.columns}
                        onColumnToggle={this.toggleColumn} />
                </Grid>
            </Toolbar>
            <DataTable
                columns={this.state.columns}
                items={this.props.items}
                onRowContextMenu={this.openItemMenuOnRowClick} />
            <Toolbar />
        </Paper>;
    }

    toggleColumn = (column: DataColumn<DataItem>) => {
        const index = this.state.columns.indexOf(column);
        const columns = this.state.columns.slice(0);
        columns.splice(index, 1, { ...column, selected: !column.selected });
        this.setState({ columns });
    }

    renderName = (item: DataItem) =>
        <Grid
            container
            alignItems="center"
            wrap="nowrap"
            spacing={16}
            onClick={() => this.props.onItemClick(item)}>
            <Grid item>
                {renderIcon(item)}
            </Grid>
            <Grid item>
                <Typography color="primary">
                    {item.name}
                </Typography>
            </Grid>
        </Grid>

    renderActions = (item: DataItem) =>
        <Grid container justify="flex-end">
            <IconButton onClick={event => this.openItemMenuOnActionsClick(event, item)}>
                <MoreVertIcon />
            </IconButton>
        </Grid>

    openItemMenuOnRowClick = (event: React.MouseEvent<HTMLElement>, item: DataItem) => {
        event.preventDefault();
        this.setState({
            contextMenu: {
                anchorEl: mockAnchorFromMouseEvent(event),
                item
            }
        });
    }

    openItemMenuOnActionsClick = (event: React.MouseEvent<HTMLElement>, item: DataItem) => {
        this.setState({
            contextMenu: {
                anchorEl: event.currentTarget,
                item
            }
        });
    }

    closeContextMenu = () => {
        this.setState({ contextMenu: {} });
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

const renderStatus = (status?: string) =>
    <Typography noWrap align="center">
        {status || "-"}
    </Typography>;

export default DataExplorer;
