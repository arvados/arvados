// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip, IconButton, Typography } from "@mui/material";
import { AddFavoriteIcon, RemoveFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { FavoritesState } from "store/favorites/favorites-reducer";
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { componentItemStyles, ComponentCssRules } from "../component-item-styles";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import classNames from "classnames";

type ToggleFavoriteActionProps = {
    isInToolbar: boolean,
    contextMenuResourceUuid: string,
    selectedResourceUuid?: string,
    favorites: FavoritesState,
    disabledButtons: Set<string>,
    onClick: () => void
}

const mapStateToProps = (state: RootState): Pick<ToggleFavoriteActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'favorites' | 'disabledButtons'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResource.selectedResourceUuid,
    favorites: state.favorites,
    disabledButtons: new Set<string>(state.multiselect.disabledButtons),
});

export const ToggleFavoriteAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: ToggleFavoriteActionProps & WithStyles<ComponentCssRules>) => {
    const { classes, onClick, isInToolbar, contextMenuResourceUuid, selectedResourceUuid, favorites, disabledButtons } = props;

    const faveResourceUuid = isInToolbar ? selectedResourceUuid : contextMenuResourceUuid;
    const isFavorite = faveResourceUuid !== undefined && favorites[faveResourceUuid] === true;
    const isDisabled = disabledButtons.has(ContextMenuActionNames.ADD_TO_FAVORITES);

    return <Tooltip title={isFavorite ? "Remove from favorites" : "Add to favorites"}>
        {props.isInToolbar ? (
            <IconButton
                data-cy='multiselect-button'
                className={classes.toolbarButton}
                disabled={isDisabled}
                onClick={onClick}>
                <ListItemIcon className={classNames(classes.toolbarIcon, isDisabled && classes.disabled)}>
                    {isFavorite
                        ? <RemoveFavoriteIcon />
                        : <AddFavoriteIcon />}
                </ListItemIcon>
            </IconButton>
        ) : (
            <ListItem
                button
                onClick={onClick}>
                <ListItemIcon>
                    {isFavorite
                        ? <RemoveFavoriteIcon />
                        : <AddFavoriteIcon />}
                </ListItemIcon>
                <ListItemText style={{ textDecoration: 'none' }}>
                {isFavorite
                        ? <Typography>Remove from favorites</Typography>
                        : <Typography>Add to favorites</Typography>}
                </ListItemText>
        </ListItem>
        )}
    </Tooltip>
}));
