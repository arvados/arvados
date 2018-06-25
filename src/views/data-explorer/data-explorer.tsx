// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RouteComponentProps } from 'react-router';
import { Project } from '../../models/project';
import { ProjectState } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from "../../components/data-table/data-table";
import DataExplorer, { DataExplorerContextActions } from "../../views-components/data-explorer/data-explorer";
import { projectExplorerItems } from "./data-explorer-selectors";
import { DataItem } from "../../views-components/data-explorer/data-item";
import { CollectionState } from "../../store/collection/collection-reducer";
import { setProjectItem } from "../../store/navigation/navigation-action";

interface DataExplorerViewDataProps {
    projects: ProjectState;
    collections: CollectionState;
}

type DataExplorerViewProps = DataExplorerViewDataProps & RouteComponentProps<{ uuid: string }> & DispatchProp;
type DataExplorerViewState = DataColumns<Project>;

class DataExplorerView extends React.Component<DataExplorerViewProps, DataExplorerViewState> {
    render() {
        console.log('VIEW!');
        const items = projectExplorerItems(
            this.props.projects.items,
            this.props.projects.currentItemId,
            this.props.collections
        );
        return (
            <DataExplorer
                items={items}
                onItemClick={this.goToItem}
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

    goToItem = (item: DataItem) => {
        this.props.dispatch<any>(setProjectItem(this.props.projects.items, item.uuid, item.kind));
    }
}

export default connect(
    (state: RootState) => ({
        projects: state.projects,
        collections: state.collections
    })
)(DataExplorerView);
