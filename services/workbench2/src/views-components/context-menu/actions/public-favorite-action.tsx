// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip } from "@mui/material";
import { PublicFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { PublicFavoritesState } from "store/public-favorites/public-favorites-reducer";

const toolbarIconClass = {
    width: '1rem',
    marginLeft: '-0.5rem',
    marginTop: '0.25rem',
}

const mapStateToProps = (state: RootState) => ({
    isPublicFavorite: state.contextMenu.resource !== undefined && state.publicFavorites[state.contextMenu.resource.uuid] === true,
    contextMenuResource: state.contextMenu.resource,
    selectedResourceUuid: state.selectedResourceUuid,
    publicFavorites: state.publicFavorites,
});

type TogglePublicFavoriteActionProps = {
    isInToolbar?: boolean;
    contextMenuResource: ContextMenuResource;
    selectedResourceUuid?: string;
    publicFavorites: PublicFavoritesState;
    onClick: () => void;
};

export const TogglePublicFavoriteAction = connect(mapStateToProps)((props: TogglePublicFavoriteActionProps) => {
    const publicFaveUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResource.uuid;
    const isPublicFavorite = publicFaveUuid !== undefined && props.publicFavorites[publicFaveUuid] === true;

    return <Tooltip title={isPublicFavorite ? "Remove from public favorites" : "Add to public favorites"}>
    <ListItem
        button
        onClick={props.onClick}>
        <ListItemIcon style={props.isInToolbar ? toolbarIconClass : {}}>
            {isPublicFavorite
                ? <PublicFavoriteIcon />
                : <PublicFavoriteIcon />}
        </ListItemIcon>
            {!props.isInToolbar &&
                <ListItemText style={{ textDecoration: 'none' }}>
                    {isPublicFavorite
                        ? <>Remove from public favorites</>
                        : <>Add to public favorites</>}
                </ListItemText>}
        </ListItem>
    </Tooltip>
});
