// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';
import { List, ListItem, ListItemText, ListItemIcon, Collapse, Typography } from "@material-ui/core";
import { SidePanelRightArrowIcon, IconType } from '../icon/icon';
import * as classnames from "classnames";

export interface SidePanelItem {
    id: string;
    name: string;
    icon: IconType;
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
        const { leftSidePanelContainer, row, list, toggableIconContainer } = classes;
        return (
            <div className={leftSidePanelContainer}>
                <List>
                    {sidePanelItems.map(it => (
                        <span key={it.name}>
                            <ListItem button className={list} onClick={() => toggleActive(it.id)} onContextMenu={this.handleRowContextMenu(it)}>
                                <span className={row}>
                                    {it.openAble ? (
                                        <i onClick={() => toggleOpen(it.id)} className={toggableIconContainer}>
                                            <ListItemIcon className={this.getToggableIconClassNames(it.open, it.active)}>
                                                {< SidePanelRightArrowIcon />}
                                            </ListItemIcon>
                                        </i>
                                    ) : null}
                                    <ListItemIcon className={this.getListItemIconClassNames(it.margin, it.active)}>
                                        {<it.icon />}
                                    </ListItemIcon>
                                    <ListItemText primary={this.renderListItemText(it.name, it.active)} />
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

    getToggableIconClassNames = (isOpen?: boolean, isActive ?: boolean) => {
        const { classes } = this.props;
        return classnames(classes.toggableIcon, {
            [classes.iconOpen]: isOpen,
            [classes.iconClose]: !isOpen,
            [classes.active]: isActive
        });
    }

    getListItemIconClassNames = (hasMargin?: boolean, isActive?: boolean) => {
        const { classes } = this.props;
        return classnames({
            [classes.hasMargin]: hasMargin,
            [classes.active]: isActive
        });
    }

    renderListItemText = (name: string, isActive?: boolean) => {
        return <Typography variant='body1' className={this.getListItemTextClassNames(isActive)}>
                {name}
            </Typography>;
    }

    getListItemTextClassNames = (isActive?: boolean) => {
        const { classes } = this.props;
        return classnames(classes.listItemText, {
            [classes.active]: isActive
        });
    }

    handleRowContextMenu = (item: SidePanelItem) =>
        (event: React.MouseEvent<HTMLElement>) =>
            item.openAble ? this.props.onContextMenu(event, item) : null

}

type CssRules = 'active' | 'listItemText' | 'row' | 'leftSidePanelContainer' | 'list' | 
    'hasMargin' | 'iconClose' | 'iconOpen' | 'toggableIconContainer' | 'toggableIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
    row: {
        display: 'flex',
        alignItems: 'center',
    },
    toggableIconContainer: {
        color: theme.palette.grey["700"],
        height: '14px',
        position: 'absolute'
    },
    toggableIcon: {
        fontSize: '14px'
    },
    listItemText: {
        fontWeight: 700
    },
    active: {
        color: theme.palette.primary.main,
    },
    hasMargin: {
        marginLeft: '18px',
    },
    iconClose: {
        transition: 'all 0.1s ease',
    },
    iconOpen: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    }
});

export default withStyles(styles)(SidePanel);