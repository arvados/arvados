// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { List, ListItem, ListItemIcon, Checkbox, Radio } from "@material-ui/core";
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import CircularProgress from '@material-ui/core/CircularProgress';
import classnames from "classnames";

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
    | 'checkbox'
    | 'childItem';

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
    },
    childItem: {
        cursor: 'pointer',
        display: 'flex',
        padding: '3px 20px',
        fontSize: '0.875rem',
        '&:hover': {
            backgroundColor: 'rgba(0, 0, 0, 0.08)',
        }
    },
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

const flatTree = (depth: number, items?: any): [] => {
    return items ? items.reduce((prev: any, next: any) => {
        const { items } = next;
        // delete next.items;
        return [
            ...prev,
            { ...next, depth },
            ...(next.open ? flatTree(depth + 1, items) : []),
        ];
    }, []) : [];
};

const getActionAndId = (event: any, initAction: string | undefined = undefined) => {
    const { nativeEvent: { target } } = event;
    let currentTarget: HTMLElement = target as HTMLElement;
    let action: string | undefined = initAction || currentTarget.dataset.action;
    let id: string | undefined = currentTarget.dataset.id;

    while (action === undefined || id === undefined) {
        currentTarget = currentTarget.parentElement as HTMLElement;

        if (!currentTarget) {
            break;
        }

        action = action || currentTarget.dataset.action;
        id = id || currentTarget.dataset.id;
    }

    return [action, id];
};

export const Tree = withStyles(styles)(
    class Component<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
        render(): ReactElement<any> {
            const level = this.props.level ? this.props.level : 0;
            const { classes, render, items, toggleItemActive, disableRipple, currentItemUuid, useRadioButtons } = this.props;
            const { list, listItem, loader, toggableIconContainer, renderContainer, childItem, active } = classes;
            const showSelection = typeof this.props.showSelection === 'function'
                ? this.props.showSelection
                : () => this.props.showSelection ? true : false;

            const { levelIndentation = 20, itemRightPadding = 20 } = this.props;

            const flatItems = (items || [])
                .map(parentItem => ({
                    ...parentItem,
                    items: flatTree(2, parentItem.items || []),
                }));

            return <List className={list}>
                {flatItems && flatItems.map((it: TreeItem<T>, idx: number) =>
                    <div key={`item/${level}/${it.id}`}>
                        <ListItem button className={listItem}
                            style={{
                                paddingLeft: (level + 1) * levelIndentation,
                                paddingRight: itemRightPadding,
                            }}
                            disableRipple={disableRipple}
                            onClick={event => toggleItemActive(event, it)}
                            selected={showSelection(it) && it.id === currentItemUuid}
                            onContextMenu={(event) => this.props.onContextMenu(event, it)}>
                            {it.status === TreeItemStatus.PENDING ?
                                <CircularProgress size={10} className={loader} /> : null}
                            <i onClick={(e) => this.handleToggleItemOpen(it, e)}
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
                            <div
                                onContextMenu={(event) => {
                                    const [action, id] = getActionAndId(event, 'CONTEXT_MENU');
                                    this.props.onContextMenu(event, { id } as any);
                                }}
                                onClick={(event) => {
                                    const [action, id] = getActionAndId(event);

                                    if (action && id) {
                                        switch(action) {
                                            case 'TOGGLE_OPEN':
                                                this.handleToggleItemOpen({ id } as any, event);
                                                break;
                                            case 'TOGGLE_ACTIVE':
                                                toggleItemActive(event, { id } as any);
                                                break;
                                            default:
                                                break;
                                        }
                                    }
                                }}
                            >
                                {
                                    it.items
                                        .map((item: any) => <div key={item.id} data-id={item.id}
                                            className={classnames(childItem, { [active]: item.active })}
                                            style={{ paddingLeft: `${item.depth * levelIndentation}px`}}>
                                            <i data-action="TOGGLE_OPEN" className={toggableIconContainer}>
                                                <ListItemIcon className={this.getToggableIconClassNames(item.open, item.active)}>
                                                    {this.getProperArrowAnimation(item.status, item.items!)}
                                                </ListItemIcon>
                                            </i>
                                            <div style={{ marginLeft: '8px' }} data-action="TOGGLE_ACTIVE" className={renderContainer}>
                                                {item.data.name}
                                            </div>
                                        </div>)
                                }
                                {/* <Tree
                                    showSelection={this.props.showSelection}
                                    items={it.items}
                                    render={render}
                                    disableRipple={disableRipple}
                                    toggleItemOpen={toggleItemOpen}
                                    toggleItemActive={toggleItemActive}
                                    level={level + 1}
                                    onContextMenu={onContextMenu}
                                    toggleItemSelection={this.props.toggleItemSelection} /> */}
                            </div>}
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

        handleCheckboxChange = (item: TreeItem<T>) => {
            const { toggleItemSelection } = this.props;
            return toggleItemSelection
                ? (event: React.MouseEvent<HTMLElement>) => {
                    event.stopPropagation();
                    toggleItemSelection(event, item);
                }
                : undefined;
        }

        handleToggleItemOpen = (item: TreeItem<T>, event: React.MouseEvent<HTMLElement>) => {
            event.stopPropagation();
            this.props.toggleItemOpen(event, item);
        }
    }
);
