// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { Collapse, Tooltip } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import StarIcon from '@mui/icons-material/Star';
import { renderIcon } from 'views-components/data-explorer/renderers';
import { loadFavoritePanel } from 'store/favorite-panel/favorite-panel-action';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { openContextMenuOnlyFromUuid } from 'store/context-menu/context-menu-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { navigateTo } from 'store/navigation/navigation-action';

type CssRules = 'root' | 'title' | 'hr' | 'list' | 'item' | 'name' | 'icon' | 'namePlate' | 'star';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    title: {
        margin: '0 1rem',
        padding: '4px',
    },
    hr: {
        marginTop: '0',
        marginBottom: '0',
    },
    list: {
        marginTop: '0.5rem',
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-start',
        width: '100%',
    },
    item: {
        width: '18rem',
        height: '3.5rem',
        margin: theme.spacing(2),
        marginTop: '0',
        padding: theme.spacing(1),
        background: '#fafafa',
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

const mapStateToProps = (state: RootState) => {
    const selection = state.dataExplorer.favoritePanel?.items || [];
    const faves = selection.map((uuid) => state.resources[uuid]);
    return {
        items: faves as GroupContentsResource[],
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    loadFavoritePanel: () => dispatch<any>(loadFavoritePanel()),
    navTo: (uuid: string) => dispatch<any>(navigateTo(uuid)),
    openContextMenu: (ev: React.MouseEvent<HTMLElement>, uuid: string) => dispatch<any>(openContextMenuOnlyFromUuid(ev, uuid)),
});

type FavePinsSectionProps = ReturnType<typeof mapStateToProps> & ReturnType<typeof mapDispatchToProps> & WithStyles<CssRules>;

export const FavePinsSection = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)(({ items, classes, loadFavoritePanel, navTo, openContextMenu }: FavePinsSectionProps) => {

        useEffect(() => {
            loadFavoritePanel();
        }, [loadFavoritePanel]);

        const [isOpen, setIsOpen] = useState(true);

        return (
            <div className={classes.root}>
                <div className={classes.title} onClick={() => setIsOpen(!isOpen)}>
                    <span>Favorites</span>
                    <ExpandChevronRight expanded={isOpen} />
                    <hr className={classes.hr} />
                </div>
                <Collapse in={isOpen}>
                        <div className={classes.list}>
                            {items.map((item) => (
                                <FavePinItem
                                    key={item.uuid}
                                    item={item}
                                    classes={classes}
                                    navTo={navTo}
                                    openContextMenu={openContextMenu}
                                />
                            ))}
                        </div>
                </Collapse>
            </div>
        )
    })
);

type FavePinItemProps = {
    item: GroupContentsResource,
    navTo: (uuid: string) => void,
    openContextMenu: (event: React.MouseEvent, uuid: string) => void
};

const FavePinItem = ({ item, openContextMenu, navTo, classes }: FavePinItemProps & WithStyles<CssRules>) => {
    console.log(">>> FavePinItem", item);

    const handleContextMenu = (event: React.MouseEvent) => {
        event.preventDefault();
        event.stopPropagation();
        openContextMenu(event, item.uuid);
    };

    return (
        <div className={classes.item} onContextMenu={handleContextMenu} onClick={() => navTo(item.uuid)}>
            <div className={classes.icon}>{renderIcon(item)}</div>
            <div className={classes.namePlate}>
                <div className={classes.name}>{item.name}</div>
            </div>
            <Tooltip title='Remove from Favorites'>
                <StarIcon className={classes.star} />
            </Tooltip>
        </div>
    );
};
