// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip, IconButton, Typography } from "@mui/material";
import { PublicFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { PublicFavoritesState } from "store/public-favorites/public-favorites-reducer";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { componentItemStyles, ComponentCssRules } from "../component-item-styles";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import classNames from "classnames";

const mapStateToProps = (state: RootState): Pick<TogglePublicFavoriteActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'publicFavorites' | 'disabledButtons'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResource.selectedResourceUuid,
    publicFavorites: state.publicFavorites,
    disabledButtons: new Set<string>(state.multiselect.disabledButtons),
});

type TogglePublicFavoriteActionProps = {
    isInToolbar: boolean;
    contextMenuResourceUuid: string;
    selectedResourceUuid?: string;
    publicFavorites: PublicFavoritesState;
    disabledButtons: Set<string>,
    onClick: () => void;
};

export const TogglePublicFavoriteAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: TogglePublicFavoriteActionProps & WithStyles<ComponentCssRules>) => {
    const { classes, onClick, isInToolbar, contextMenuResourceUuid, selectedResourceUuid, publicFavorites, disabledButtons } = props;

    const publicFaveUuid = isInToolbar ? selectedResourceUuid : contextMenuResourceUuid;
    const isPublicFavorite = publicFaveUuid !== undefined && publicFavorites[publicFaveUuid] === true;
    const isDisabled = disabledButtons.has(ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES);

    return <Tooltip title={isPublicFavorite ? "Remove from public favorites" : "Add to public favorites"}>
        {isInToolbar ? (
            <IconButton
                data-cy='multiselect-button'
                className={classes.toolbarButton}
                disabled={isDisabled}
                onClick={onClick}>
                <ListItemIcon className={classNames(classes.toolbarIcon, isDisabled && classes.disabled)}>
                    {isPublicFavorite
                        ? <PublicFavoriteIcon />
                        : <PublicFavoriteIcon />}
                </ListItemIcon>
            </IconButton>
        ) : (
            <ListItem
                button
                onClick={onClick}>
                <ListItemIcon>
                    {isPublicFavorite
                        ? <PublicFavoriteIcon />
                        : <PublicFavoriteIcon />}
                </ListItemIcon>
                <ListItemText style={{ textDecoration: 'none' }}>
                    {isPublicFavorite
                        ? <Typography>Remove from public favorites</Typography>
                        : <Typography>Add to public favorites</Typography>}
                </ListItemText>
            </ListItem>
        )}
    </Tooltip>
}));
