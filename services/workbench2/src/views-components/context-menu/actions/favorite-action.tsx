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

type ToggleFavoriteActionProps = {
    isInToolbar: boolean,
    contextMenuResourceUuid: string,
    selectedResourceUuid?: string,
    favorites: FavoritesState,
    onClick: () => void
}

const mapStateToProps = (state: RootState): Pick<ToggleFavoriteActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'favorites'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResourceUuid,
    favorites: state.favorites,
});

export const ToggleFavoriteAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: ToggleFavoriteActionProps & WithStyles<ComponentCssRules>) => {
    const faveResourceUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const isFavorite = faveResourceUuid !== undefined && props.favorites[faveResourceUuid] === true;

    return <Tooltip title={isFavorite ? "Remove from favorites" : "Add to favorites"}>
        {props.isInToolbar ? (
            <IconButton
                className={props.classes.toolbarButton}
                onClick={props.onClick}>
                <ListItemIcon className={props.classes.toolbarIcon}>
                    {isFavorite
                        ? <RemoveFavoriteIcon />
                        : <AddFavoriteIcon />}
                </ListItemIcon>
            </IconButton>
        ) : (
            <ListItem
                button
                onClick={props.onClick}>
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
