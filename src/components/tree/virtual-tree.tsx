// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as classnames from "classnames";
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import { FixedSizeList, ListChildComponentProps } from "react-window";
import AutoSizer from "react-virtualized-auto-sizer";

import { ArvadosTheme } from '~/common/custom-theme';
import { TreeItem, TreeProps, TreeItemStatus } from './tree';
import { ListItem, Radio, Checkbox, CircularProgress, ListItemIcon } from '@material-ui/core';
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
    | 'virtualizedList';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    list: {
        padding: '3px 0px',
    },
    virtualizedList: {
        height: '200px',
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

// export const RowA = <T, _>(items: TreeItem<T>[], render:any) => (index: number) => {
//     return <div>
//         {render(items[index])}
//     </div>;
// };

// For some reason, on TSX files it isn't accepted just one generic param, so
// I'm using <T, _> as a workaround.
export const Row =  <T, _>(itemList: TreeItem<T>[], render: any) => withStyles(styles)(
    (props: React.PropsWithChildren<ListChildComponentProps> & TreeProps<T> & WithStyles<CssRules>) => {
        const { index, style } = props;
        const it = itemList[index];
        const level = it.level || 0;
        const { classes, toggleItemActive, disableRipple, currentItemUuid, useRadioButtons } = props;
        const { listItem, loader, toggableIconContainer, renderContainer } = classes;
        const { levelIndentation = 20, itemRightPadding = 20 } = props;

        const showSelection = typeof props.showSelection === 'function'
            ? props.showSelection
            : () => props.showSelection ? true : false;

        const handleRowContextMenu = (item: TreeItem<T>) =>
            (event: React.MouseEvent<HTMLElement>) =>
                props.onContextMenu(event, item);

        const handleToggleItemOpen = (item: TreeItem<T>) =>
            (event: React.MouseEvent<HTMLElement>) => {
                event.stopPropagation();
                props.toggleItemOpen(event, item);
            };

        const getToggableIconClassNames = (isOpen?: boolean, isActive?: boolean) => {
            const { iconOpen, iconClose, active, toggableIcon } = props.classes;
            return classnames(toggableIcon, {
                [iconOpen]: isOpen,
                [iconClose]: !isOpen,
                [active]: isActive
            });
        };

        const isSidePanelIconNotNeeded = (status: string, itemCount: number) => {
            return status === TreeItemStatus.PENDING ||
                (status === TreeItemStatus.LOADED && itemCount === 0);
        };

        const getProperArrowAnimation = (status: string, itemCount: number) => {
            return isSidePanelIconNotNeeded(status, itemCount) ? <span /> : <SidePanelRightArrowIcon style={{ fontSize: '14px' }} />;
        };

        const handleCheckboxChange = (item: TreeItem<T>) => {
            const { toggleItemSelection } = props;
            return toggleItemSelection
                ? (event: React.MouseEvent<HTMLElement>) => {
                    event.stopPropagation();
                    toggleItemSelection(event, item);
                }
                : undefined;
        };

        return <div style={style}>
            <ListItem button className={listItem}
                style={{
                    paddingLeft: (level + 1) * levelIndentation,
                    paddingRight: itemRightPadding,
                }}
                disableRipple={disableRipple}
                onClick={event => toggleItemActive(event, it)}
                selected={showSelection(it) && it.id === currentItemUuid}
                onContextMenu={handleRowContextMenu(it)}>
                {it.status === TreeItemStatus.PENDING ?
                    <CircularProgress size={10} className={loader} /> : null}
                <i onClick={handleToggleItemOpen(it)}
                    className={toggableIconContainer}>
                    <ListItemIcon className={getToggableIconClassNames(it.open, it.active)}>
                        {getProperArrowAnimation(it.status, it.itemCount!)}
                    </ListItemIcon>
                </i>
                {showSelection(it) && !useRadioButtons &&
                    <Checkbox
                        checked={it.selected}
                        className={classes.checkbox}
                        color="primary"
                        onClick={handleCheckboxChange(it)} />}
                {showSelection(it) && useRadioButtons &&
                    <Radio
                        checked={it.selected}
                        className={classes.checkbox}
                        color="primary" />}
                <div className={renderContainer}>
                    {render(it, level)}
                </div>
            </ListItem>
        </div>;
    });

export const VirtualList = <T, _>(height: number, width: number, items: TreeItem<T>[], render: any) =>
    <FixedSizeList
        height={height}
        itemCount={items.length}
        itemSize={30}
        width={width}
    >
        {Row(items, render)}
    </FixedSizeList>;

export const VirtualTree = withStyles(styles)(
    class Component<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
        render(): ReactElement<any> {
            const { items, render } = this.props;

            return <div className={this.props.classes.virtualizedList}><AutoSizer>
                {({ height, width }) => {
                    return VirtualList(height, width, items || [], render);
                }}
            </AutoSizer></div>;
        }
    }
);
