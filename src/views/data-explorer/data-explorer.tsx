// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RouteComponentProps } from 'react-router';
import { Project } from '../../models/project';
import { ProjectState } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { push } from 'react-router-redux';
import { DataColumns } from "../../components/data-table/data-table";
import DataExplorer, { DataExplorerContextActions } from "../../views-components/data-explorer/data-explorer";
import { projectExplorerItems } from "./data-explorer-selectors";
import { DataItem } from "../../views-components/data-explorer/data-item";
import { CollectionState } from "../../store/collection/collection-reducer";
import { ResourceKind } from "../../models/resource";
import projectActions from "../../store/project/project-action";
import { getCollectionList } from "../../store/collection/collection-action";

interface DataExplorerViewDataProps {
    projects: ProjectState;
    collections: CollectionState;
}

type DataExplorerViewProps = DataExplorerViewDataProps & RouteComponentProps<{ uuid: string }> & DispatchProp;
type DataExplorerViewState = DataColumns<Project>;

class DataExplorerView extends React.Component<DataExplorerViewProps, DataExplorerViewState> {
    render() {
        const treeItemId = this.props.match.params.uuid;
        const items = projectExplorerItems(this.props.projects, treeItemId, this.props.collections);
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
        // FIXME: Unify project tree switch action
        this.props.dispatch(push(item.url));
        if (item.type === ResourceKind.PROJECT || item.type === ResourceKind.LEVEL_UP) {
            this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(item.uuid));
        }
        this.props.dispatch<any>(getCollectionList(item.uuid));
    }
}

export default connect(
    (state: RootState) => ({
        projects: state.projects,
        collections: state.collections
    })
)(DataExplorerView);
