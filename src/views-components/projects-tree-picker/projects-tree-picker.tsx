// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { TreeItem, TreeItemStatus } from '~/components/tree/tree';
import { ProjectResource } from "~/models/project";
import { treePickerActions } from "~/store/tree-picker/tree-picker-actions";
import { ListItemTextIcon } from "~/components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon, InputIcon, IconType, CollectionIcon } from '~/components/icon/icon';
import { loadProject, loadCollection } from '~/store/tree-picker/tree-picker-actions';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { CollectionDirectory, CollectionFile, CollectionFileType } from '~/models/collection-file';
import { ResourceKind } from '~/models/resource';
import { TreePickerProps, TreePicker } from "~/views-components/tree-picker/tree-picker";

export interface ProjectsTreePickerRootItem {
    id: string;
    name: string;
}

type ProjectsTreePickerItem = ProjectsTreePickerRootItem | GroupContentsResource | CollectionDirectory | CollectionFile;
type PickedTreePickerProps = Pick<TreePickerProps<ProjectsTreePickerItem>, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen' | 'toggleItemSelection'>;

export interface ProjectsTreePickerDataProps {
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
    rootItemIcon: IconType;
    loadRootItem: (item: TreeItem<ProjectsTreePickerRootItem>, pickerId: string, includeCollections?: boolean, inlcudeFiles?: boolean) => void;
}

export interface ProjectsTreePickerActionProps {
}

export type ProjectsTreePickerProps = ProjectsTreePickerDataProps & ProjectsTreePickerActionProps;

const mapStateToProps = (_: any, { pickerId, rootItemIcon }: ProjectsTreePickerProps) => ({
    render: renderTreeItem(rootItemIcon),
    pickerId,
});

const mapDispatchToProps = (dispatch: Dispatch, { loadRootItem, includeCollections, includeFiles }: ProjectsTreePickerProps): PickedTreePickerProps => ({
    onContextMenu: () => { return; },
    toggleItemActive: (_, { id }, pickerId) => {
        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id, pickerId }));
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
                loadRootItem(item as TreeItem<ProjectsTreePickerRootItem>, pickerId, includeCollections, includeFiles);
            }
        } else if (status === TreeItemStatus.LOADED) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
        }
    },
    toggleItemSelection: (_, { id }, pickerId) => {
        dispatch<any>(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECTION({ id, pickerId }));
    },
});

export const ProjectsTreePicker = connect(mapStateToProps, mapDispatchToProps)(TreePicker);

const getProjectPickerIcon = ({ data }: TreeItem<ProjectsTreePickerItem>, rootIcon: IconType): IconType => {
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
                return InputIcon;
            default:
                return ProjectIcon;
        }
    } else {
        return rootIcon;
    }
};

const renderTreeItem = (rootItemIcon: IconType) => (item: TreeItem<ProjectResource>) =>
    <ListItemTextIcon
        icon={getProjectPickerIcon(item, rootItemIcon)}
        name={item.data.name}
        isActive={item.active}
        hasMargin={true} />;
