// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect, Dispatch, DispatchProp } from "react-redux";
import { RootState } from "../../store/store";
import actions from "../../store/context-menu/context-menu-actions";
import ContextMenu, { ContextMenuAction, ContextMenuProps } from "../../components/context-menu/context-menu";
import { createAnchorAt } from "../../components/popover/helpers";
import projectActions from "../../store/project/project-action";
import { ContextMenuResource } from "../../store/context-menu/context-menu-reducer";


type DataProps = Pick<ContextMenuProps, "anchorEl" | "actions"> & { resource?: ContextMenuResource };
const mapStateToProps = (state: RootState): DataProps => {
    const { position, resource } = state.contextMenu;
    return {
        anchorEl: resource ? createAnchorAt(position) : undefined,
        actions: contextMenuActions,
        resource
    };
};

type ActionProps = Pick<ContextMenuProps, "onClose"> & {onActionClick: (action: ContextMenuAction, resource?: ContextMenuResource) => void};
const mapDispatchToProps = (dispatch: Dispatch): ActionProps => ({
    onClose: () => {
        dispatch(actions.CLOSE_CONTEXT_MENU());
    },
    onActionClick: (action: ContextMenuAction, resource?: ContextMenuResource) => {
        dispatch(actions.CLOSE_CONTEXT_MENU());
        if (resource) {
            if (action.name === "New project") {
                dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
            }
        }
    }
});

const mergeProps = ({ resource, ...dataProps }: DataProps, actionProps: ActionProps): ContextMenuProps => ({
    ...dataProps,
    ...actionProps,
    onActionClick: (action: ContextMenuAction) => {
        actionProps.onActionClick(action, resource);
    }
});

export default connect(mapStateToProps, mapDispatchToProps, mergeProps)(ContextMenu);

const contextMenuActions = [[{
    icon: "fas fa-plus fa-fw",
    name: "New project"
}, {
    icon: "fas fa-users fa-fw",
    name: "Share"
}, {
    icon: "fas fa-sign-out-alt fa-fw",
    name: "Move to"
}, {
    icon: "fas fa-star fa-fw",
    name: "Add to favourite"
}, {
    icon: "fas fa-edit fa-fw",
    name: "Rename"
}, {
    icon: "fas fa-copy fa-fw",
    name: "Make a copy"
}, {
    icon: "fas fa-download fa-fw",
    name: "Download"
}], [{
    icon: "fas fa-trash-alt fa-fw",
    name: "Remove"
}
]];