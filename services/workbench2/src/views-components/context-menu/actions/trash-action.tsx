// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip , Typography, IconButton } from "@mui/material";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { GroupResource } from "models/group";
import { RootState } from "store/store";
import { ResourcesState, getResource } from "store/resources/resources";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { componentItemStyles, ComponentCssRules } from "../component-item-styles";

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

export const ToggleTrashAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: ToggleTrashActionProps & WithStyles<ComponentCssRules>) => {
    const trashResourceUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const isTrashed = getResource<GroupResource>(trashResourceUuid)(props.resources)?.isTrashed;

    return (
        <Tooltip title={isTrashed ? "Restore" : "Move to trash"}>
            {props.isInToolbar ? (
                <IconButton
                    className={props.classes.toolbarButton}
                    onClick={props.onClick}>
                    <ListItemIcon className={props.classes.toolbarIcon}>
                        {isTrashed
                            ? <RestoreFromTrashIcon />
                            : <TrashIcon />}
                    </ListItemIcon>
                </IconButton>
            ) : (
            <ListItem button
                onClick={props.onClick}>
                <ListItemIcon>
                    {isTrashed
                        ? <RestoreFromTrashIcon/>
                        : <TrashIcon/>}
                </ListItemIcon>
                    <ListItemText style={{ textDecoration: 'none' }}>
                        <Typography>
                            {isTrashed ? "Restore" : "Move to trash"}
                        </Typography>
                    </ListItemText>
            </ListItem >
            )}
        </Tooltip>
    )
}));
