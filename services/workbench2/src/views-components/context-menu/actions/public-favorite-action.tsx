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

const mapStateToProps = (state: RootState): Pick<TogglePublicFavoriteActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'publicFavorites'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResourceUuid,
    publicFavorites: state.publicFavorites,
});

type TogglePublicFavoriteActionProps = {
    isInToolbar: boolean;
    contextMenuResourceUuid: string;
    selectedResourceUuid?: string;
    publicFavorites: PublicFavoritesState;
    onClick: () => void;
};

export const TogglePublicFavoriteAction = connect(mapStateToProps)(withStyles(componentItemStyles)((props: TogglePublicFavoriteActionProps & WithStyles<ComponentCssRules>) => {
    const publicFaveUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const isPublicFavorite = publicFaveUuid !== undefined && props.publicFavorites[publicFaveUuid] === true;

    return <Tooltip title={isPublicFavorite ? "Remove from public favorites" : "Add to public favorites"}>
        {props.isInToolbar ? (
            <IconButton
                className={props.classes.toolbarButton}
                onClick={props.onClick}>
                <ListItemIcon className={props.classes.toolbarIcon}>
                    {isPublicFavorite
                        ? <PublicFavoriteIcon />
                        : <PublicFavoriteIcon />}
                </ListItemIcon>
            </IconButton>
        ) : (
            <ListItem
                button
                onClick={props.onClick}>
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
