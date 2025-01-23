// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip } from "@mui/material";
import { PublicFavoriteIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { PublicFavoritesState } from "store/public-favorites/public-favorites-reducer";

const toolbarIconClass = {
    width: '1rem',
    marginLeft: '-0.5rem',
    marginTop: '0.25rem',
}

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

export const TogglePublicFavoriteAction = connect(mapStateToProps)((props: TogglePublicFavoriteActionProps) => {
    const publicFaveUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
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
