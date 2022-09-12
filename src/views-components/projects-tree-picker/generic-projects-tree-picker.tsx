// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { isEqual } from 'lodash/fp';
import { TreeItem, TreeItemStatus } from 'components/tree/tree';
import { ProjectResource } from "models/project";
import { treePickerActions } from "store/tree-picker/tree-picker-actions";
import { ListItemTextIcon } from "components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon, FileInputIcon, IconType, CollectionIcon } from 'components/icon/icon';
import { loadProject, loadCollection } from 'store/tree-picker/tree-picker-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { CollectionDirectory, CollectionFile, CollectionFileType } from 'models/collection-file';
import { ResourceKind } from 'models/resource';
import { TreePickerProps, TreePicker } from "views-components/tree-picker/tree-picker";
import { LinkResource } from "models/link";

export interface ProjectsTreePickerRootItem {
    id: string;
    name: string;
}

export type ProjectsTreePickerItem = ProjectsTreePickerRootItem | GroupContentsResource | CollectionDirectory | CollectionFile | LinkResource;
type PickedTreePickerProps = Pick<TreePickerProps<ProjectsTreePickerItem>, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen' | 'toggleItemSelection'>;

export interface ProjectsTreePickerDataProps {
    includeCollections?: boolean;
    includeFiles?: boolean;
    rootItemIcon: IconType;
    showSelection?: boolean;
    relatedTreePickers?: string[];
    disableActivation?: string[];
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    loadRootItem: (item: TreeItem<ProjectsTreePickerRootItem>, pickerId: string,
         includeCollections?: boolean, includeFiles?: boolean, options?: { showOnlyOwned: boolean, showOnlyWritable: boolean }) => void;
}

export type ProjectsTreePickerProps = ProjectsTreePickerDataProps & Partial<PickedTreePickerProps>;

const mapStateToProps = (_: any, { rootItemIcon, showSelection }: ProjectsTreePickerProps) => ({
    render: renderTreeItem(rootItemIcon),
    showSelection: isSelectionVisible(showSelection),
});

const mapDispatchToProps = (dispatch: Dispatch, { loadRootItem, includeCollections, includeFiles, relatedTreePickers, options, ...props }: ProjectsTreePickerProps): PickedTreePickerProps => ({
    onContextMenu: () => { return; },
    toggleItemActive: (event, item, pickerId) => {

        const { disableActivation = [] } = props;
        if (disableActivation.some(isEqual(item.id))) {
            return;
        }

        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: item.id, pickerId, relatedTreePickers }));
        if (props.toggleItemActive) {
            props.toggleItemActive(event, item, pickerId);
        }
    },
    toggleItemOpen: (_, item, pickerId) => {
        const { id, data, status } = item;
        if (status === TreeItemStatus.INITIAL) {
            if ('kind' in data) {
                dispatch<any>(
                    data.kind === ResourceKind.COLLECTION
                        ? loadCollection(id, pickerId)
                        : loadProject({ id, pickerId, includeCollections, includeFiles })
                );
            } else if (!('type' in data) && loadRootItem) {
                loadRootItem(item as TreeItem<ProjectsTreePickerRootItem>, pickerId, includeCollections, includeFiles, options);
            }
        } else if (status === TreeItemStatus.LOADED) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
        }
    },
    toggleItemSelection: (event, item, pickerId) => {
        dispatch<any>(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECTION({ id: item.id, pickerId }));
        if (props.toggleItemSelection) {
            props.toggleItemSelection(event, item, pickerId);
        }
    },
});

export const ProjectsTreePicker = connect(mapStateToProps, mapDispatchToProps)(TreePicker);

const getProjectPickerIcon = ({ data }: TreeItem<ProjectsTreePickerItem>, rootIcon: IconType): IconType => {
    if ('headKind' in data) {
        switch (data.headKind) {
            case ResourceKind.COLLECTION:
                return CollectionIcon;
            default:
                return ProjectIcon;
        }
    }
    if ('kind' in data) {
        switch (data.kind) {
            case ResourceKind.COLLECTION:
                return CollectionIcon;
            default:
                return ProjectIcon;
        }
    } else if ('type' in data) {
        switch (data.type) {
            case CollectionFileType.FILE:
                return FileInputIcon;
            default:
                return ProjectIcon;
        }
    } else {
        return rootIcon;
    }
};

const isSelectionVisible = (shouldBeVisible?: boolean) =>
    ({ status, items }: TreeItem<ProjectsTreePickerItem>): boolean => {
        if (shouldBeVisible) {
            if (items && items.length > 0) {
                return items.every(isSelectionVisible(shouldBeVisible));
            }
            return status === TreeItemStatus.LOADED;
        }
        return false;
    };

const renderTreeItem = (rootItemIcon: IconType) => (item: TreeItem<ProjectResource>) =>
    <ListItemTextIcon
        icon={getProjectPickerIcon(item, rootItemIcon)}
        name={item.data.name}
        isActive={item.active}
        hasMargin={true} />;
