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
import { ProjectsTreePickerItem, ProjectsTreePickerRootItem } from 'store/tree-picker/tree-picker-middleware';
import { ResourceKind } from 'models/resource';
import { TreePickerProps, TreePicker } from "views-components/tree-picker/tree-picker";
import { CollectionFileType } from 'models/collection-file';


type PickedTreePickerProps = Pick<TreePickerProps<ProjectsTreePickerItem>, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen' | 'toggleItemSelection'>;

export interface ProjectsTreePickerDataProps {
    cascadeSelection: boolean;
    includeCollections?: boolean;
    includeDirectories?: boolean;
    includeFiles?: boolean;
    rootItemIcon: IconType;
    showSelection?: boolean;
    relatedTreePickers?: string[];
    disableActivation?: string[];
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    loadRootItem: (item: TreeItem<ProjectsTreePickerRootItem>, pickerId: string,
        includeCollections?: boolean, includeDirectories?: boolean, includeFiles?: boolean, options?: { showOnlyOwned: boolean, showOnlyWritable: boolean }) => void;
}

export type ProjectsTreePickerProps = ProjectsTreePickerDataProps & Partial<PickedTreePickerProps>;

const mapStateToProps = (_: any, { rootItemIcon, showSelection, cascadeSelection }: ProjectsTreePickerProps) => ({
    render: renderTreeItem(rootItemIcon),
    showSelection: isSelectionVisible(showSelection, cascadeSelection),
});

const mapDispatchToProps = (dispatch: Dispatch, { loadRootItem, includeCollections, includeDirectories, includeFiles, relatedTreePickers, options, ...props }: ProjectsTreePickerProps): PickedTreePickerProps => ({
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
                        ? loadCollection(id, pickerId, includeDirectories, includeFiles)
                        : loadProject({ id, pickerId, includeCollections, includeDirectories, includeFiles, options })
                );
            } else if (!('type' in data) && loadRootItem) {
                loadRootItem(item as TreeItem<ProjectsTreePickerRootItem>, pickerId, includeCollections, includeDirectories, includeFiles, options);
            }
        } else if (status === TreeItemStatus.LOADED) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
        }
    },
    toggleItemSelection: (event, item, pickerId) => {
        dispatch<any>(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECTION({ id: item.id, pickerId, cascade: props.cascadeSelection }));
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

const isSelectionVisible = (shouldBeVisible: boolean | undefined, cascadeSelection: boolean) =>
    ({ status, items, data }: TreeItem<ProjectsTreePickerItem>): boolean => {
        if (shouldBeVisible) {
            if (!cascadeSelection && 'kind' in data && data.kind === ResourceKind.COLLECTION) {
                // In non-casecade mode collections are selectable without being loaded
                return true;
            } else if (items && items.length > 0) {
                return items.every(isSelectionVisible(shouldBeVisible, cascadeSelection));
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
