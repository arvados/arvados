// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useMemo, ReactElement, JSXElementConstructor } from 'react';
import { DoubleRightArrows } from 'components/icon/icon';
import classnames from 'classnames';
import { IconButton, Menu, MenuItem, StyleRulesCallback, Tooltip, WithStyles, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'inOverflowMenu' | 'openMenuButton' | 'menu' | 'menuItem' | 'menuElement';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    inOverflowMenu: {
        '&:hover': {
            backgroundColor: 'transparent',
        },
    },
    openMenuButton: {
        right: '10px',
    },
    menu: {
        marginLeft: 0,
    },
    menuItem: {
        '&:hover': {
            backgroundColor: 'white',
        },
        marginTop: 0,
        paddingTop: 0,
        paddingLeft: '1rem',
        height: '2.5rem',
    },
    menuElement: {
        width: '2rem',
    }
});

export type OverflowChild = ReactElement<{ className: string; }, string | JSXElementConstructor<any>>

type OverflowMenuProps = {
    children: OverflowChild[]
    className: string
    visibilityMap: {}
}

export const OverflowMenu = withStyles(styles)((props: OverflowMenuProps & WithStyles<CssRules>) => {
    const { children, className, visibilityMap, classes } = props;
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);
    const handleClick = (event) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const shouldShowMenu = useMemo(() => Object.values(visibilityMap).some((v) => v === false), [visibilityMap]);
    if (!shouldShowMenu) {
        return null;
    }
    return (
        <div className={className}>
            <Tooltip title="More Options" disableFocusListener>
                <IconButton
                    aria-label='more'
                    aria-controls='long-menu'
                    aria-haspopup='true'
                    onClick={handleClick}
                    className={classes.openMenuButton}
                >
                        <DoubleRightArrows />
                </IconButton>
            </Tooltip>
            <Menu
                id='long-menu'
                anchorEl={anchorEl}
                keepMounted
                open={open}
                onClose={handleClose}
                disableAutoFocusItem
                className={classes.menu}
            >
                {React.Children.map(children, (child: any) => {
                    if (!visibilityMap[child.props['data-targetid']]) {
                        return (
                            <MenuItem
                                key={child}
                                onClick={handleClose}
                                className={classes.menuItem}
                            >
                                {React.cloneElement(child, {
                                    className: classnames(classes.menuElement),
                                })}
                            </MenuItem>
                        );
                    }
                    return null;
                })}
            </Menu>
        </div>
    );
});
