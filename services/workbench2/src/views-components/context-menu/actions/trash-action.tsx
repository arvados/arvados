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
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import classNames from "classnames";
import { matchTrashRoute } from "routes/routes";

const mapStateToProps = (state: RootState): Pick<ToggleTrashActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'resources' | 'disabledButtons' | 'pathname'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResource.selectedResourceUuid,
    resources: state.resources,
    disabledButtons: new Set<string>(state.multiselect.disabledButtons),
    pathname: state.router.location?.pathname,
});

type ToggleTrashActionProps = {
    isInToolbar: boolean;
    contextMenuResourceUuid: string;
    selectedResourceUuid: string;
    resources: ResourcesState
    disabledButtons: Set<string>,
    pathname: string | undefined;
    onClick: () => void;
};

export const ToggleTrashAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: ToggleTrashActionProps & WithStyles<ComponentCssRules>) => {
    const { classes, onClick, isInToolbar, contextMenuResourceUuid, selectedResourceUuid, resources, disabledButtons } = props;

    const currentPathIsTrash = matchTrashRoute(props.pathname || "");
    const trashResourceUuid = isInToolbar ? selectedResourceUuid : contextMenuResourceUuid;
    const isTrashed = getResource<GroupResource>(trashResourceUuid)(resources)?.isTrashed || currentPathIsTrash;
    const isDisabled = disabledButtons.has(ContextMenuActionNames.MOVE_TO_TRASH);

    return isInToolbar ? (
            <Tooltip title={isTrashed ? "Restore" : "Move to trash"}>
                <IconButton
                    data-cy='multiselect-button'
                    className={classes.toolbarButton}
                    disabled={isDisabled}
                    onClick={onClick}>
                    <ListItemIcon className={classNames(classes.toolbarIcon, isDisabled && classes.disabled)}>
                        {isTrashed
                            ? <RestoreFromTrashIcon />
                            : <TrashIcon />}
                    </ListItemIcon>
                </IconButton>
            </Tooltip>
            ) : (
            <ListItem button
                onClick={onClick}>
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
        )
}));
