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

type CssRules = 'root' | 'title' | 'list' | 'item' | 'name' | 'icon' | 'star';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    title: {
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        borderRadius: '4px',
        marginLeft: '1rem',
        padding: '4px',
        '&:hover': {
            background: 'lightgray',
        },
    },
    list: {
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-start',
        width: '100%',
    },
    item: {
        width: '100px',
        height: '100px',
        margin: theme.spacing(2),
        marginTop: '0.5rem',
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
    const selection = (state.dataExplorer.favoritePanel?.items || []);
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
            items ? <div className={classes.root}>
                <span className={classes.title} onClick={() => setIsOpen(!isOpen)}>Favorites</span>
                {isOpen ? <Collapse in={isOpen}>
                    <div className={classes.list}>
                        {items.map((item) => (
                            <FavePinItem
                            key={item.uuid}
                            item={item}
                            classes={classes}
                            />
                        ))}
                    </div>
                </Collapse> : <div style={{margin: '1rem'}}><hr/></div>}
            </div> : <div>Loading...</div>
        );
    })
);

const FavePinItem = ({ item, classes }: { item: any } & WithStyles<CssRules>) => {
    return (
        <div className={classes.item}>
            <div className={classes.icon}>{renderIcon(item)}</div>
            <div className={classes.name}>{item.name}</div>
            <Tooltip title="Remove from Favorites">
                <StarIcon className={classes.star} />
            </Tooltip>
        </div>
    );
};
