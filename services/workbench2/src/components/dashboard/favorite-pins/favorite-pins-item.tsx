// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Tooltip } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { ArvadosTheme } from 'common/custom-theme';
import StarIcon from '@mui/icons-material/Star';
import { renderIcon } from 'views-components/data-explorer/renderers';
import { loadFavoritePanel } from 'store/favorite-panel/favorite-panel-action';
import { openContextMenuOnlyFromUuid } from 'store/context-menu/context-menu-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { navigateTo } from 'store/navigation/navigation-action';
import { toggleFavorite } from 'store/favorites/favorites-actions';

type CssRules = 'item' | 'name' | 'icon' | 'namePlate' | 'star';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    item: {
        height: '3.5rem',
        marginTop: '0',
        padding: theme.spacing(1),
        backgroundColor: theme.palette.background.paper,
        borderRadius: '8px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'relative',
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
        textAlign: 'center',
        overflow: 'hidden',
        boxSizing: 'border-box',
        cursor: 'pointer',
        '&:hover': {
            background: theme.palette.grey[200],
        },
    },
    name: {
        width: '100%',
        fontSize: '0.875rem',
        textAlign: 'left',
        lineHeight: '1.2',
        maxHeight: '2.5rem',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        display: '-webkit-box',
        WebkitLineClamp: 2,
        WebkitBoxOrient: 'vertical',
    },
    icon: {
        color: theme.customs.colors.grey700,
        marginRight: '0.5rem',
    },
    namePlate: {
        width: '80%',
        display: 'flex',
        flexDirection: 'column',
    },
    star: {
        fontSize: '1.25rem',
        color: theme.customs.colors.grey700,
        marginLeft: '0.5rem',
    },
});

const mapDispatchToProps = (dispatch: Dispatch): Omit<FavePinItemProps, 'item'> => ({
    navTo: (uuid: string) => dispatch<any>(navigateTo(uuid)),
    toggleFavorite: (item: GroupContentsResource) => dispatch<any>(toggleFavorite({ uuid: item.uuid, name: item.name })).then(() => dispatch<any>(loadFavoritePanel())),
    openContextMenu: (ev: React.MouseEvent<HTMLElement>, uuid: string) => dispatch<any>(openContextMenuOnlyFromUuid(ev, uuid)),
});

type FavePinItemProps = {
    item: GroupContentsResource,
    navTo: (uuid: string) => void,
    toggleFavorite: (item: GroupContentsResource) => void,
    openContextMenu: (event: React.MouseEvent, uuid: string) => void
};

export const FavePinItem = connect(null, mapDispatchToProps)(
    withStyles(styles)(({ item, openContextMenu, navTo, toggleFavorite, classes }: FavePinItemProps & WithStyles<CssRules>) => {

    const handleContextMenu = (event: React.MouseEvent) => {
        event.preventDefault();
        event.stopPropagation();
        openContextMenu(event, item.uuid);
    };

    const handleToggleFavorite = (event: React.MouseEvent) => {
        event.preventDefault();
        event.stopPropagation();
        toggleFavorite(item);
    };

    return (
        <div data-cy='favorite-pin'
            className={classes.item}
            onContextMenu={handleContextMenu}
            onClick={() => navTo(item.uuid)}
            >
            <div className={classes.icon}>{renderIcon(item)}</div>
            <div className={classes.namePlate}>
                <div className={classes.name}>{item.name}</div>
            </div>
            <Tooltip title='Remove from Favorites' onClick={handleToggleFavorite}>
                <StarIcon data-cy={`${item.uuid}-star`} className={classes.star} />
            </Tooltip>
        </div>
    );
}));
