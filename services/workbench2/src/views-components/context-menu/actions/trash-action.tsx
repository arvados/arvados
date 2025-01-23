// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip } from "@mui/material";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { GroupResource } from "models/group";
import { RootState } from "store/store";
import { ResourcesState, getResource } from "store/resources/resources";

const toolbarIconClass = {
    width: '1rem',
    marginLeft: '-0.5rem',
    marginTop: '0.25rem',
};

const mapStateToProps = (state: RootState): Pick<ToggleTrashActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'resources'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResourceUuid,
    resources: state.resources,
});

type ToggleTrashActionProps = {
    isInToolbar: boolean;
    contextMenuResourceUuid: string;
    selectedResourceUuid: string;
    resources: ResourcesState
    onClick: () => void;
};

export const ToggleTrashAction = connect(mapStateToProps)((props: ToggleTrashActionProps) => {
    const trashResourceUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const isTrashed = getResource<GroupResource>(trashResourceUuid)(props.resources)?.isTrashed;

    return (
        <Tooltip title={isTrashed ? "Restore" : "Move to trash"}>
            <ListItem button
                onClick={props.onClick}>
                <ListItemIcon style={props.isInToolbar ? toolbarIconClass : {}}>
                    {isTrashed
                        ? <RestoreFromTrashIcon/>
                        : <TrashIcon/>}
                </ListItemIcon>
                {!props.isInToolbar &&
                    <ListItemText style={{ textDecoration: 'none' }}>
                        {isTrashed ? "Restore" : "Move to trash"}
                    </ListItemText>}
            </ListItem >
        </Tooltip>
    )
});
