// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import { Collapse } from '@mui/material';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import StarIcon from '@mui/icons-material/Star';
import Tooltip from '@mui/material/Tooltip';
import { renderIcon } from 'views-components/data-explorer/renderers';
import { loadFavoritePanel } from 'store/favorite-panel/favorite-panel-action';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';

type CssRules = 'root' | 'title' | 'hr' | 'list' | 'item' | 'name' | 'icon' | 'star';

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
        width: '100px',
        height: '100px',
        margin: theme.spacing(2),
        marginTop: '0',
        padding: theme.spacing(1),
        background: '#fafafa',
        borderRadius: '8px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center', // Center contents
        position: 'relative',
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
        textAlign: 'center',
        overflow: 'hidden',
        boxSizing: 'border-box',
        cursor: 'pointer',
        '&:hover': {
            background: 'lightgray',
        },
    },
    name: {
        fontSize: '0.875rem',
        textAlign: 'center',
        lineHeight: '1.2',
        maxHeight: '2.4rem', // Ensures it only takes two lines
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        display: '-webkit-box',
        WebkitLineClamp: 2, // Restricts to two lines
        WebkitBoxOrient: 'vertical',
    },
    icon: {
        color: theme.customs.colors.grey700,
        marginTop: '1rem',
        flex: '1 0 auto', // Uncomment if you have another element above the icon
    },
    star: {
        fontSize: '1.25rem',
        position: 'absolute',
        top: '5px',
        right: '5px',
        color: theme.customs.colors.grey700,
    },
});

const mapStateToProps = (state: RootState) => {
    const selection = state.dataExplorer.favoritePanel?.items || [];
    const faves = selection.map((uuid) => state.resources[uuid]);
    return {
        items: faves,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    loadFavoritePanel: () => dispatch<any>(loadFavoritePanel()),
});

export const FavePinsSection = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)(({ items, classes, loadFavoritePanel }: { items: any[] } & WithStyles<CssRules> & { loadFavoritePanel: () => void }) => {
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
                                />
                            ))}
                        </div>
                </Collapse>
            </div>
        )
    })
);

const FavePinItem = ({ item, classes }: { item: any } & WithStyles<CssRules>) => {
    return (
        <div className={classes.item}>
            <div className={classes.icon}>{renderIcon(item)}</div>
            <div className={classes.name}>{item.name}</div>
            <Tooltip title='Remove from Favorites'>
                <StarIcon className={classes.star} />
            </Tooltip>
        </div>
    );
};
