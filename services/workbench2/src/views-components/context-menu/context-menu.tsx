// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { contextMenuActions, ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenu as ContextMenuComponent, ContextMenuProps, ContextMenuItem } from "components/context-menu/context-menu";
import { createAnchorAt } from "components/popover/helpers";
import { ContextMenuActionSet, ContextMenuAction } from "./context-menu-action-set";
import { Dispatch } from "redux";
import { memoize } from "lodash";
import { sortByProperty } from "common/array-utils";

type DataProps = Pick<ContextMenuProps, "anchorEl" | "items" | "open"> & { resource?: ContextMenuResource };

const mapStateToProps = (state: RootState): DataProps => {
    const { open, position, resource } = state.contextMenu;

    const filteredItems = getMenuActionSet(resource).map(group =>
        group.filter(item => {
            if (resource && item.filters) {
                // Execute all filters on this item, every returns true IFF all filters return true
                return item.filters.every(filter => filter(state, resource));
            } else {
                return true;
            }
        })
    );

    return {
        anchorEl: resource ? createAnchorAt(position) : undefined,
        items: filteredItems,
        open,
        resource,
    };
};

type ActionProps = Pick<ContextMenuProps, "onClose"> & { onItemClick: (item: ContextMenuItem, resource?: ContextMenuResource) => void };
const mapDispatchToProps = (dispatch: Dispatch): ActionProps => ({
    onClose: () => {
        dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
    },
    onItemClick: (action: ContextMenuAction, resource?: ContextMenuResource) => {
        dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
        if (resource) {
            action.execute(dispatch, [resource]);
        }
    },
});

const handleItemClick = memoize(
    (resource: DataProps["resource"], onItemClick: ActionProps["onItemClick"]): ContextMenuProps["onItemClick"] =>
        item => {
            onItemClick(item, { ...resource, fromContextMenu: true } as ContextMenuResource);
        }
);

const mergeProps = ({ resource, ...dataProps }: DataProps, actionProps: ActionProps): ContextMenuProps => ({
    ...dataProps,
    ...actionProps,
    onItemClick: handleItemClick(resource, actionProps.onItemClick),
});

export const ContextMenu = connect(mapStateToProps, mapDispatchToProps, mergeProps)(ContextMenuComponent);

const menuActionSets = new Map<string, ContextMenuActionSet>();

export const addMenuActionSet = (name: string, itemSet: ContextMenuActionSet) => {
    const sorted = itemSet.map(items => items.sort(sortByProperty("name")));
    menuActionSets.set(name, sorted);
};

const emptyActionSet: ContextMenuActionSet = [];
const getMenuActionSet = (resource?: ContextMenuResource): ContextMenuActionSet =>
    resource ? menuActionSets.get(resource.menuKind) || emptyActionSet : emptyActionSet;

export enum ContextMenuKind {
    API_CLIENT_AUTHORIZATION = "ApiClientAuthorization",
    ROOT_PROJECT = "RootProject",
    PROJECT = "Project",
    FILTER_GROUP = "FilterGroup",
    READONLY_PROJECT = "ReadOnlyProject",
    FROZEN_PROJECT = "FrozenProject",
    FROZEN_PROJECT_ADMIN = "FrozenProjectAdmin",
    PROJECT_ADMIN = "ProjectAdmin",
    FILTER_GROUP_ADMIN = "FilterGroupAdmin",
    RESOURCE = "Resource",
    FAVORITE = "Favorite",
    TRASH = "Trash",
    COLLECTION_FILES = "CollectionFiles",
    COLLECTION_FILES_MULTIPLE = "CollectionFilesMultiple",
    READONLY_COLLECTION_FILES = "ReadOnlyCollectionFiles",
    READONLY_COLLECTION_FILES_MULTIPLE = "ReadOnlyCollectionFilesMultiple",
    COLLECTION_FILES_NOT_SELECTED = "CollectionFilesNotSelected",
    COLLECTION_FILE_ITEM = "CollectionFileItem",
    COLLECTION_DIRECTORY_ITEM = "CollectionDirectoryItem",
    READONLY_COLLECTION_FILE_ITEM = "ReadOnlyCollectionFileItem",
    READONLY_COLLECTION_DIRECTORY_ITEM = "ReadOnlyCollectionDirectoryItem",
    COLLECTION = "Collection",
    COLLECTION_ADMIN = "CollectionAdmin",
    READONLY_COLLECTION = "ReadOnlyCollection",
    OLD_VERSION_COLLECTION = "OldVersionCollection",
    TRASHED_COLLECTION = "TrashedCollection",
    PROCESS = "Process",
    RUNNING_PROCESS_ADMIN = "RunningProcessAdmin",
    PROCESS_ADMIN = "ProcessAdmin",
    RUNNING_PROCESS_RESOURCE = "RunningProcessResource",
    PROCESS_RESOURCE = "ProcessResource",
    READONLY_PROCESS_RESOURCE = "ReadOnlyProcessResource",
    PROCESS_LOGS = "ProcessLogs",
    REPOSITORY = "Repository",
    SSH_KEY = "SshKey",
    VIRTUAL_MACHINE = "VirtualMachine",
    KEEP_SERVICE = "KeepService",
    USER = "User",
    GROUPS = "Group",
    GROUP_MEMBER = "GroupMember",
    PERMISSION_EDIT = "PermissionEdit",
    LINK = "Link",
    WORKFLOW = "Workflow",
    READONLY_WORKFLOW = "ReadOnlyWorkflow",
    SEARCH_RESULTS = "SearchResults",
}
