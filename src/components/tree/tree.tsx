// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import Collapse from "@material-ui/core/Collapse/Collapse";
import CircularProgress from '@material-ui/core/CircularProgress';
import * as classnames from "classnames";
import { ListItemIcon } from '@material-ui/core/';

import { ArvadosTheme } from '../../common/custom-theme';
import { SidePanelRightArrowIcon } from '../icon/icon';

type CssRules = 'list' | 'active' | 'loader' | 'toggableIconContainer' | 'iconClose' | 'iconOpen' | 'toggableIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    list: {
        paddingBottom: '3px',
        paddingTop: '3px',
    },
    loader: {
        position: 'absolute',
        transform: 'translate(0px)',
        top: '3px'
    },
    toggableIconContainer: {
        color: theme.palette.grey["700"],
        height: '14px',
        position: 'absolute'
    },
    toggableIcon: {
        fontSize: '14px'
    },
    active: {
        color: theme.palette.primary.main,
    },
    iconClose: {
        transition: 'all 0.1s ease',
    },
    iconOpen: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    }
});

export enum TreeItemStatus {
    Initial,
    Pending,
    Loaded
}

export interface TreeItem<T> {
    data: T;
    id: string;
    open: boolean;
    active: boolean;
    status: TreeItemStatus;
    toggled?: boolean;
    items?: Array<TreeItem<T>>;
}

interface TreeProps<T> {
    items?: Array<TreeItem<T>>;
    render: (item: TreeItem<T>, level?: number) => ReactElement<{}>;
    toggleItemOpen: (id: string, status: TreeItemStatus) => void;
    toggleItemActive: (id: string, status: TreeItemStatus) => void;
    level?: number;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
}

export const Tree = withStyles(styles)(
    class Component<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
        render(): ReactElement<any> {
            const level = this.props.level ? this.props.level : 0;
            const { classes, render, toggleItemOpen, items, toggleItemActive, onContextMenu } = this.props;
            const { list, loader, toggableIconContainer } = classes;
            return <List component="div" className={list}>
                {items && items.map((it: TreeItem<T>, idx: number) =>
                    <div key={`item/${level}/${idx}`}>
                        <ListItem button className={list} style={{ paddingLeft: (level + 1) * 20 }}
                            onClick={() => toggleItemActive(it.id, it.status)}
                            onContextMenu={this.handleRowContextMenu(it)}>
                            {it.status === TreeItemStatus.Pending ?
                                <CircularProgress size={10} className={loader} /> : null}
                            <i onClick={() => this.props.toggleItemOpen(it.id, it.status)}
                                className={toggableIconContainer}>
                                <ListItemIcon className={this.getToggableIconClassNames(it.open, it.active)}>
                                    {it.toggled && it.items && it.items.length === 0 ? <span /> : <SidePanelRightArrowIcon />}
                                </ListItemIcon>
                            </i>
                            {render(it, level)}
                        </ListItem>
                        {it.items && it.items.length > 0 &&
                            <Collapse in={it.open} timeout="auto" unmountOnExit>
                                <Tree
                                    items={it.items}
                                    render={render}
                                    toggleItemOpen={toggleItemOpen}
                                    toggleItemActive={toggleItemActive}
                                    level={level + 1}
                                    onContextMenu={onContextMenu} />
                            </Collapse>}
                    </div>)}
            </List>;
        }

        getToggableIconClassNames = (isOpen?: boolean, isActive?: boolean) => {
            const { iconOpen, iconClose, active, toggableIcon } = this.props.classes;
            return classnames(toggableIcon, {
                [iconOpen]: isOpen,
                [iconClose]: !isOpen,
                [active]: isActive
            });
        }

        handleRowContextMenu = (item: TreeItem<T>) =>
            (event: React.MouseEvent<HTMLElement>) =>
                this.props.onContextMenu(event, item)
    }
);
