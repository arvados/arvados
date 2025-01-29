// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip, IconButton, Typography } from "@mui/material";
import { FreezeIcon, UnfreezeIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { resourceIsFrozen } from "common/frozen-resources";
import { getResource } from "store/resources/resources";
import { GroupResource } from "models/group";
import { memoize } from "lodash";
import { ResourcesState } from "store/resources/resources";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { componentItemStyles, ComponentCssRules } from "../component-item-styles";

type ToggleLockActionProps = {
    isInToolbar: boolean;
    selectedResourceUuid: string;
    contextMenuResourceUuid: string,
    resources: ResourcesState,
    onClick: () => void;
};

const mapStateToProps = (state: RootState): Pick<ToggleLockActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'resources'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResourceUuid,
    resources: state.resources,
});

export const ToggleLockAction = connect(mapStateToProps)(withStyles(componentItemStyles)(memoize((props: ToggleLockActionProps & WithStyles<ComponentCssRules>) => {
    const lockResourceUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const resource = getResource<GroupResource>(lockResourceUuid)(props.resources);
    const isLocked = resource ? resourceIsFrozen(resource, props.resources) : false;

    return (
        <Tooltip title={isLocked ? "Unfreeze project" : "Freeze project"}>
            {props.isInToolbar ? (
                <IconButton
                className={props.classes.toolbarButton}
                onClick={props.onClick}>
                <ListItemIcon className={props.classes.toolbarIcon}>
                        {isLocked
                            ? <UnfreezeIcon />
                            : <FreezeIcon />}
                    </ListItemIcon>
                </IconButton>
            ) : (
            <ListItem button onClick={props.onClick} data-cy="toggle-lock-action">
                <ListItemIcon>
                    {isLocked
                        ? <UnfreezeIcon />
                        : <FreezeIcon />}
                </ListItemIcon>
                    <ListItemText style={{ textDecoration: 'none' }}>
                        {isLocked
                            ? <Typography>Unfreeze project</Typography>
                            : <Typography>Freeze project</Typography>}
                    </ListItemText>
            </ListItem>)}
        </Tooltip>
    );
})));