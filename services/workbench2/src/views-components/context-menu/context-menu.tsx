// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { contextMenuActions, ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenu as ContextMenuComponent, ContextMenuProps, ContextMenuItem } from "components/context-menu/context-menu";
import { ContextMenuAction } from "./context-menu-action-set";
import { Dispatch } from "redux";
import { memoize } from "lodash";

type DataProps = Pick<ContextMenuProps, "contextMenu"> & { resource?: ContextMenuResource };

const mapStateToProps = (state: RootState): DataProps => {
    return {
        contextMenu: state.contextMenu,
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




