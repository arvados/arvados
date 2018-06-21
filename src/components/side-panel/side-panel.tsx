// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import ListItemText from "@material-ui/core/ListItemText/ListItemText";
import ListItemIcon from '@material-ui/core/ListItemIcon';
import Collapse from "@material-ui/core/Collapse/Collapse";

import { Typography } from '@material-ui/core';

export interface SidePanelItem {
    id: string;
    name: string;
    icon: string;
    active?: boolean;
    open?: boolean;
}

interface SidePanelProps {
    toggleSidePanelOpen: (id: string) => void;
    toggleSidePanelActive: (id: string) => void;
    sidePanelItems: SidePanelItem[];
}

class SidePanel extends React.Component<SidePanelProps & WithStyles<CssRules>> {
    render(): ReactElement<any> {
        const { classes, toggleSidePanelOpen, toggleSidePanelActive, sidePanelItems } = this.props;
        const { listItemText, leftSidePanelContainer, row, list, icon, projectIcon, active, activeArrow, inactiveArrow, arrowTransition, arrowRotate } = classes;
        return (
            <div className={leftSidePanelContainer}>
                <List>
                    {sidePanelItems.map(it => (
                        <span key={it.name}>
                            <ListItem button className={list} onClick={() => toggleSidePanelActive(it.id)}>
                                <span className={row}>
                                    {it.name === "Projects" ? <i onClick={() => toggleSidePanelOpen(it.id)} className={`${it.active ? activeArrow : inactiveArrow} 
                                        ${it.open ? `fas fa-caret-down ${arrowTransition}` : `fas fa-caret-down ${arrowRotate}`}`} /> : null}
                                    <ListItemIcon className={it.active ? active : ''}>
                                        <i className={`${it.icon} ${icon} ${it.name === "Projects" ? projectIcon : ''}`} />
                                    </ListItemIcon>
                                    <ListItemText className={listItemText} primary={<Typography className={it.active ? active : ''}>{it.name}</Typography>} />
                                </span>
                            </ListItem>
                            {it.name === "Projects" ? (
                                <Collapse in={it.open} timeout="auto" unmountOnExit>
                                    {this.props.children}
                                </Collapse>) : null}
                        </span>
                    ))}
                </List>
            </div>
        );
    }
}

type CssRules = 'active' | 'listItemText' | 'row' | 'leftSidePanelContainer' | 'list' | 'icon' | 'projectIcon' |
    'activeArrow' | 'inactiveArrow' | 'arrowRotate' | 'arrowTransition';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    active: {
        color: '#4285F6',
    },
    listItemText: {
        padding: '0px',
    },
    row: {
        display: 'flex',
        alignItems: 'center',
    },
    activeArrow: {
        color: '#4285F6',
        position: 'absolute',
    },
    inactiveArrow: {
        position: 'absolute',
    },
    arrowTransition: {
        transition: 'all 0.1s ease',
    },
    arrowRotate: {
        transition: 'all 0.1s ease',
        transform: 'rotate(-90deg)',
    },
    leftSidePanelContainer: {
        overflowY: 'auto',
        minWidth: '240px',
        whiteSpace: 'nowrap',
        marginTop: '38px',
        display: 'flex',
        flexGrow: 1,
    },
    list: {
        paddingBottom: '5px',
        paddingTop: '5px',
        paddingLeft: '14px',
        minWidth: '240px',
    },
    icon: {
        minWidth: '20px',
    },
    projectIcon: {
        marginLeft: '17px',
    }
});

export default withStyles(styles)(SidePanel);