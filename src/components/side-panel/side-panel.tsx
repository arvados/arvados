// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import { List, ListItem, ListItemText, ListItemIcon, Collapse, Typography } from "@material-ui/core";
import { SidePanelRightArrowIcon } from '../icon/icon';
import * as classnames from "classnames";

export interface SidePanelItem {
    id: string;
    name: string;
    icon: (className?: string) => React.ReactElement<any>;
    active?: boolean;
    open?: boolean;
    margin?: boolean;
    openAble?: boolean;
}

interface SidePanelProps {
    toggleOpen: (id: string) => void;
    toggleActive: (id: string) => void;
    sidePanelItems: SidePanelItem[];
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: SidePanelItem) => void;
}

class SidePanel extends React.Component<SidePanelProps & WithStyles<CssRules>> {
    render(): ReactElement<any> {
        const { classes, toggleOpen, toggleActive, sidePanelItems, children } = this.props;
        const { listItemText, leftSidePanelContainer, row, list, icon, projectIconMargin, active, iconArrowContainer } = classes;
        return (
            <div className={leftSidePanelContainer}>
                <List>
                    {sidePanelItems.map(it => (
                        <span key={it.name}>
                            <ListItem button className={list} onClick={() => toggleActive(it.id)} onContextMenu={this.handleRowContextMenu(it)}>
                                <span className={row}>
                                    {it.openAble ? (
                                        <i onClick={() => toggleOpen(it.id)} className={iconArrowContainer}>
                                            {SidePanelRightArrowIcon(this.getIconClassNames(it.open, it.active))}
                                        </i>
                                    ) : null}
                                    <ListItemIcon className={it.active ? active : ''}>
                                        {it.icon(`${icon} ${it.margin ? projectIconMargin : ''}`)}
                                    </ListItemIcon>
                                    <ListItemText className={listItemText} 
                                        primary={renderListItemText(it.name, active, it.active)} />
                                </span>
                            </ListItem>
                            {it.openAble ? (
                                <Collapse in={it.open} timeout="auto" unmountOnExit>
                                    {children}
                                </Collapse>
                            ) : null}
                        </span>
                    ))}
                </List>
            </div>
        );
    }

    getIconClassNames = (itemOpen ?: boolean, itemActive ?: boolean) => {
        const { classes } = this.props;
        return classnames(classes.iconArrow, {
            [classes.iconOpen]: itemOpen,
            [classes.iconClose]: !itemOpen,
            [classes.active]: itemActive
        });
    }

    handleRowContextMenu = (item: SidePanelItem) =>
        (event: React.MouseEvent<HTMLElement>) =>
            item.openAble ? this.props.onContextMenu(event, item) : null

}

const renderListItemText = (itemName: string, active: string, itemActive?: boolean) =>
    <Typography className={itemActive ? active : ''}>{itemName}</Typography>;

type CssRules = 'active' | 'listItemText' | 'row' | 'leftSidePanelContainer' | 'list' | 'icon' | 
    'projectIconMargin' | 'iconClose' | 'iconOpen' | 'iconArrowContainer' | 'iconArrow';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    active: {
        color: theme.palette.primary.main,
    },
    listItemText: {
        padding: '0px',
    },
    row: {
        display: 'flex',
        alignItems: 'center',
    },
    iconArrowContainer: {
        color: theme.palette.grey["700"],
        height: '14px',
        position: 'absolute'
    },
    iconArrow: {
        fontSize: '14px'
    },
    iconClose: {
        transition: 'all 0.1s ease',
    },
    iconOpen: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    },
    leftSidePanelContainer: {
        overflowY: 'auto',
        minWidth: '240px',
        whiteSpace: 'nowrap',
        marginTop: '52px',
        display: 'flex',
        flexGrow: 1,
    },
    list: {
        padding: '5px 0px 5px 14px',
        minWidth: '240px',
    },
    icon: {
        fontSize: '20px'
    },
    projectIconMargin: {
        marginLeft: '17px',
    }
});

export default withStyles(styles)(SidePanel);