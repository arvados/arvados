// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { List, ListItem, ListItemIcon, Collapse, Checkbox, Radio } from "@material-ui/core";
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import CircularProgress from '@material-ui/core/CircularProgress';
import * as classnames from "classnames";

import { ArvadosTheme } from '~/common/custom-theme';
import { SidePanelRightArrowIcon } from '../icon/icon';

type CssRules = 'list'
    | 'listItem'
    | 'active'
    | 'loader'
    | 'toggableIconContainer'
    | 'iconClose'
    | 'renderContainer'
    | 'iconOpen'
    | 'toggableIcon'
    | 'checkbox';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    list: {
        padding: '3px 0px'
    },
    listItem: {
        padding: '3px 0px',
    },
    loader: {
        position: 'absolute',
        transform: 'translate(0px)',
        top: '3px'
    },
    toggableIconContainer: {
        color: theme.palette.grey["700"],
        height: '14px',
        width: '14px',
    },
    toggableIcon: {
        fontSize: '14px'
    },
    renderContainer: {
        flex: 1
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
    },
    checkbox: {
        width: theme.spacing.unit * 3,
        height: theme.spacing.unit * 3,
        margin: `0 ${theme.spacing.unit}px`,
        padding: 0,
        color: theme.palette.grey["500"],
    }
});

export enum TreeItemStatus {
    INITIAL = 'initial',
    PENDING = 'pending',
    LOADED = 'loaded'
}

export interface TreeItem<T> {
    data: T;
    id: string;
    open: boolean;
    active: boolean;
    selected?: boolean;
    status: TreeItemStatus;
    items?: Array<TreeItem<T>>;
}

export interface TreeProps<T> {
    disableRipple?: boolean;
    currentItemUuid?: string;
    items?: Array<TreeItem<T>>;
    level?: number;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    render: (item: TreeItem<T>, level?: number) => ReactElement<{}>;
    showSelection?: boolean | ((item: TreeItem<T>) => boolean);
    levelIndentation?: number;
    itemRightPadding?: number;
    toggleItemActive: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    toggleItemOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    toggleItemSelection?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;

    /**
     * When set to true use radio buttons instead of checkboxes for item selection.
     * This does not guarantee radio group behavior (i.e item mutual exclusivity).
     * Any item selection logic must be done in the toggleItemActive callback prop.
     */
    useRadioButtons?: boolean;
}

export const Tree = withStyles(styles)(
    class Component<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
        render(): ReactElement<any> {
            const level = this.props.level ? this.props.level : 0;
            const { classes, render, toggleItemOpen, items, toggleItemActive, onContextMenu, disableRipple, currentItemUuid, useRadioButtons } = this.props;
            const { list, listItem, loader, toggableIconContainer, renderContainer } = classes;
            const showSelection = typeof this.props.showSelection === 'function'
                ? this.props.showSelection
                : () => this.props.showSelection ? true : false;

            const { levelIndentation = 20, itemRightPadding = 20 } = this.props;

            return <List className={list}>
                {items && items.map((it: TreeItem<T>, idx: number) =>
                    <div key={`item/${level}/${idx}`}>
                        <ListItem button className={listItem}
                            style={{
                                paddingLeft: (level + 1) * levelIndentation,
                                paddingRight: itemRightPadding,
                            }}
                            disableRipple={disableRipple}
                            onClick={event => toggleItemActive(event, it)}
                            selected={showSelection(it) && it.id === currentItemUuid}
                            onContextMenu={this.handleRowContextMenu(it)}>
                            {it.status === TreeItemStatus.PENDING ?
                                <CircularProgress size={10} className={loader} /> : null}
                            <i onClick={this.handleToggleItemOpen(it)}
                                className={toggableIconContainer}>
                                <ListItemIcon className={this.getToggableIconClassNames(it.open, it.active)}>
                                    {this.getProperArrowAnimation(it.status, it.items!)}
                                </ListItemIcon>
                            </i>
                            {showSelection(it) && !useRadioButtons &&
                                <Checkbox
                                    checked={it.selected}
                                    className={classes.checkbox}
                                    color="primary"
                                    onClick={this.handleCheckboxChange(it)} />}
                            {showSelection(it) && useRadioButtons &&
                                <Radio
                                    checked={it.selected}
                                    className={classes.checkbox}
                                    color="primary" />}
                            <div className={renderContainer}>
                                {render(it, level)}
                            </div>
                        </ListItem>
                        {it.items && it.items.length > 0 &&
                            <Collapse in={it.open} timeout="auto" unmountOnExit>
                                <Tree
                                    showSelection={this.props.showSelection}
                                    items={it.items}
                                    render={render}
                                    disableRipple={disableRipple}
                                    toggleItemOpen={toggleItemOpen}
                                    toggleItemActive={toggleItemActive}
                                    level={level + 1}
                                    onContextMenu={onContextMenu}
                                    toggleItemSelection={this.props.toggleItemSelection} />
                            </Collapse>}
                    </div>)}
            </List>;
        }

        getProperArrowAnimation = (status: string, items: Array<TreeItem<T>>) => {
            return this.isSidePanelIconNotNeeded(status, items) ? <span /> : <SidePanelRightArrowIcon style={{ fontSize: '14px' }} />;
        }

        isSidePanelIconNotNeeded = (status: string, items: Array<TreeItem<T>>) => {
            return status === TreeItemStatus.PENDING ||
                (status === TreeItemStatus.LOADED && !items) ||
                (status === TreeItemStatus.LOADED && items && items.length === 0);
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

        handleCheckboxChange = (item: TreeItem<T>) => {
            const { toggleItemSelection } = this.props;
            return toggleItemSelection
                ? (event: React.MouseEvent<HTMLElement>) => {
                    event.stopPropagation();
                    toggleItemSelection(event, item);
                }
                : undefined;
        }

        handleToggleItemOpen = (item: TreeItem<T>) => (event: React.MouseEvent<HTMLElement>) => {
            event.stopPropagation();
            this.props.toggleItemOpen(event, item);
        }
    }
);
