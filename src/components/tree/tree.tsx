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
import { ArvadosTheme } from '../../common/custom-theme';

type CssRules = 'list' | 'activeArrow' | 'inactiveArrow' | 'arrowRotate' | 'arrowTransition' | 'loader' | 'arrowVisibility';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    list: {
        paddingBottom: '3px',
        paddingTop: '3px',
    },
    activeArrow: {
        color: theme.palette.primary.main,
        position: 'absolute',
    },
    inactiveArrow: {
        color: theme.palette.grey["700"],
        position: 'absolute',
    },
    arrowTransition: {
        transition: 'all 0.1s ease',
    },
    arrowRotate: {
        transition: 'all 0.1s ease',
        transform: 'rotate(-90deg)',
    },
    arrowVisibility: {
        opacity: 0,
    },
    loader: {
        position: 'absolute',
        transform: 'translate(0px)',
        top: '3px'
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
            const { list, inactiveArrow, activeArrow, loader } = classes;
            return <List component="div" className={list}>
                {items && items.map((it: TreeItem<T>, idx: number) =>
                    <div key={`item/${level}/${idx}`}>
                        <ListItem button className={list} style={{ paddingLeft: (level + 1) * 20 }}
                                  onClick={() => toggleItemActive(it.id, it.status)}
                                  onContextMenu={this.handleRowContextMenu(it)}>
                            {it.status === TreeItemStatus.Pending ?
                                <CircularProgress size={10} className={loader}/> : null}
                            {it.toggled && it.items && it.items.length === 0 ? null : this.renderArrow(it.status, it.active ? activeArrow : inactiveArrow, it.open, it.id)}
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
                                onContextMenu={onContextMenu}/>
                        </Collapse>}
                    </div>)}
            </List>;
        }

        renderArrow(status: TreeItemStatus, arrowClass: string, open: boolean, id: string) {
            const { arrowTransition, arrowVisibility, arrowRotate } = this.props.classes;
            return <i onClick={() => this.props.toggleItemOpen(id, status)}
                      className={`
                        ${arrowClass}
                        ${status === TreeItemStatus.Pending ? arrowVisibility : ''}
                        ${open ? `fas fa-caret-down ${arrowTransition}` : `fas fa-caret-down ${arrowRotate}`}`}/>;
        }

        handleRowContextMenu = (item: TreeItem<T>) =>
            (event: React.MouseEvent<HTMLElement>) =>
                this.props.onContextMenu(event, item)
    }
);
