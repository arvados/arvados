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
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import classNames from "classnames";

type ToggleLockActionProps = {
    isInToolbar: boolean;
    selectedResourceUuid: string;
    contextMenuResourceUuid: string,
    resources: ResourcesState,
    disabledButtons: Set<string>,
    onClick: () => void;
};

const mapStateToProps = (state: RootState): Pick<ToggleLockActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'resources' | 'disabledButtons'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResource.selectedResourceUuid,
    resources: state.resources,
    disabledButtons: new Set<string>(state.multiselect.disabledButtons),
});

export const ToggleLockAction = connect(mapStateToProps)(withStyles(componentItemStyles)(memoize((props: ToggleLockActionProps & WithStyles<ComponentCssRules>) => {
    const { classes, onClick, isInToolbar, contextMenuResourceUuid, selectedResourceUuid, resources, disabledButtons } = props;

    const lockResourceUuid = isInToolbar ? selectedResourceUuid : contextMenuResourceUuid;
    const resource = getResource<GroupResource>(lockResourceUuid)(resources);
    const isLocked = resource ? resourceIsFrozen(resource, resources) : false;
    const isDisabled = disabledButtons.has(ContextMenuActionNames.FREEZE_PROJECT);

    return isInToolbar ? (
            <Tooltip title={isLocked ? "Unfreeze project" : "Freeze project"}>
                <IconButton
                data-cy='multiselect-button'
                className={classes.toolbarButton}
                disabled={isDisabled}
                onClick={onClick}>
                <ListItemIcon className={classNames(classes.toolbarIcon, isDisabled && classes.disabled)}>
                        {isLocked
                            ? <UnfreezeIcon />
                            : <FreezeIcon />}
                    </ListItemIcon>
                </IconButton>
            </Tooltip>
            ) : (
            <ListItem button onClick={onClick} data-cy="toggle-lock-action">
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
            </ListItem>
    );
})));