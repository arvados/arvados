// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useCallback, useState } from 'react';
import { List, ListItem, ListItemIcon, Checkbox, Radio, Collapse } from "@mui/material";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CollectionIcon, DefaultIcon, DirectoryIcon, FileIcon, ProjectIcon, ProcessIcon, FilterGroupIcon, FreezeIcon } from 'components/icon/icon';
import { ReactElement } from "react";
import CircularProgress from '@mui/material/CircularProgress';
import classnames from "classnames";
import { getNodeChildrenIds, Tree, getNode, initTreeNode, createTree } from 'models/tree';
import { ArvadosTheme } from 'common/custom-theme';
import { SidePanelRightArrowIcon } from '../icon/icon';
import { ResourceKind } from 'models/resource';
import { GroupClass } from 'models/group';
import { SidePanelTreeCategory } from 'store/side-panel-tree/side-panel-tree-actions';
import { kebabCase, isEqual } from 'lodash';
import { Resource } from 'models/resource';
import { ResourcesState } from 'store/resources/resources';
import { TreePicker } from 'store/tree-picker/tree-picker';

type CssRules = 'list'
              | 'listItem'
              | 'childLi'
              | 'childItemName'
              | 'active'
              | 'loader'
              | 'toggableIconContainer'
              | 'iconClose'
              | 'renderContainer'
              | 'iconOpen'
              | 'toggableIcon'
              | 'checkbox'
              | 'childItem'
              | 'childItemIcon'
              | 'frozenIcon'
              | 'indentSpacer'
              | 'itemWeightLight'
              | 'itemWeightDark';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
        marginBottom: '0.4rem',
    },
    toggableIcon: {
        fontSize: '14px',
        minWidth: '14px',
    },
    renderContainer: {
        overflow: 'hidden',
        flex: 1
    },
    iconClose: {
        transition: 'all 0.1s ease',
    },
    iconOpen: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    },
    checkbox: {
        width: theme.spacing(3),
        height: theme.spacing(3),
        margin: `0 ${theme.spacing(1)}`,
        padding: 0,
        color: theme.palette.grey["500"],
    },
    childItem: {
        cursor: 'pointer',
        display: 'flex',
        padding: '3px 20px',
        fontSize: '0.875rem',
        alignItems: 'center',
        '&:hover': {
            backgroundColor: 'rgba(0, 0, 0, 0.08)',
        }
    },
    childLi: {
        display: 'flex',
        alignItems: 'center',
    },
    childItemName: {
        fontSize: '0.875rem',
    },
    childItemIcon: {
        marginLeft: '8px',
        marginRight: '16px',
        color: 'rgba(0, 0, 0, 0.54)',
    },
    active: {
        color: theme.palette.primary.main,
    },
    itemWeightLight: {
        color: theme.customs.colors.greyL,
    },
    itemWeightDark: {
        color: "black",
    },
    frozenIcon: {
        fontSize: 20,
        color: theme.palette.grey["600"],
        marginLeft: '10px',
    },
    indentSpacer: {
        width: '0.25rem'
    }
});

export enum TreeItemStatus {
    INITIAL = 'INITIAL',
    PENDING = 'PENDING',
    LOADED = 'LOADED'
}

export interface TreeItem<T> {
    data: T;
    depth?: number;
    id: string;
    open: boolean;
    active: boolean;
    selected?: boolean;
    initialState?: boolean;
    indeterminate?: boolean;
    flatTree?: boolean;
    status: TreeItemStatus;
    items?: Array<TreeItem<T>>;
    isFrozen?: boolean;
}

export interface TreeProps<T> {
    tree?: Tree<T>;
    pickerId?: string;
    treePicker?: TreePicker;
    resources?: ResourcesState;
    currentItemUuid?: string;
    items?: Array<TreeItem<T>>;
    level?: number;
    itemsMap?: Map<string, TreeItem<T>>;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    render: (item: TreeItem<T>, level?: number) => ReactElement<{}>;
    showSelection?: boolean | ((item: TreeItem<T>) => boolean);
    levelIndentation?: number;
    itemRightPadding?: number;
    toggleItemActive: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    toggleItemOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    toggleItemSelection?: (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>) => void;
    selectedRef?: (node: HTMLDivElement | null) => void;

    /**
     * When set to true use radio buttons instead of checkboxes for item selection.
     * This does not guarantee radio group behavior (i.e item mutual exclusivity).
     * Any item selection logic must be done in the toggleItemActive callback prop.
     */
    useRadioButtons?: boolean;
}

export enum TreeItemWeight {
    NORMAL = 0,
    LIGHT = 1,
    DARK = 2,
};

export interface TreeItemWithWeight {
    weight?: TreeItemWeight;
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

const isInFavoritesTree = (item: TreeItem<any>): boolean => {
    return item.id === SidePanelTreeCategory.FAVORITES || item.id === SidePanelTreeCategory.PUBLIC_FAVORITES;
}

interface FlatTreeProps {
    it: TreeItem<any>;
    levelIndentation: number;
    onContextMenu: Function;
    handleToggleItemOpen: Function;
    toggleItemActive: Function;
    getToggableIconClassNames: Function;
    getProperArrowAnimation: Function;
    itemsMap?: Map<string, TreeItem<any>>;
    classes: any;
    showSelection: any;
    useRadioButtons?: boolean;
    handleCheckboxChange: Function;
    selectedRef?: (node: HTMLDivElement | null) => void;
}

const FLAT_TREE_ACTIONS = {
    toggleOpen: 'TOGGLE_OPEN',
    contextMenu: 'CONTEXT_MENU',
    toggleActive: 'TOGGLE_ACTIVE',
};

const ItemIcon = React.memo(({ type, kind, headKind, active, groupClass, classes }: any) => {
    let Icon = ProjectIcon;

    if (groupClass === GroupClass.FILTER) {
        Icon = FilterGroupIcon;
    }

    if (type) {
        switch (type) {
            case 'directory':
                Icon = DirectoryIcon;
                break;
            case 'file':
                Icon = FileIcon;
                break;
            default:
                Icon = DefaultIcon;
        }
    }

    if (kind) {
        if(kind === ResourceKind.LINK && headKind) kind = headKind;
        switch (kind) {
            case ResourceKind.COLLECTION:
                Icon = CollectionIcon;
                break;
            case ResourceKind.CONTAINER_REQUEST:
                Icon = ProcessIcon;
                break;
            default:
                break;
        }
    }

    return <Icon className={classnames({ [classes.active]: active }, classes.childItemIcon)} />;
});

const FlatTree = (props: FlatTreeProps) =>
    <div
        onContextMenu={(event) => {
            const id = getActionAndId(event, FLAT_TREE_ACTIONS.contextMenu)[1];
            props.onContextMenu(event, { id } as any);
        }}
        onClick={(event) => {
            const [action, id] = getActionAndId(event);

            if (action && id) {
                const item = props.itemsMap ? props.itemsMap[id] : { id };

                switch (action) {
                    case FLAT_TREE_ACTIONS.toggleOpen:
                        props.handleToggleItemOpen(item as any, event);
                        break;
                    case FLAT_TREE_ACTIONS.toggleActive:
                        props.toggleItemActive(event, item as any);
                        break;
                    default:
                        break;
                }
            }
        }}
    >
        {
            (props.it.items || [])
                .map((item: any, index: number) => <div key={item.id || index} data-id={item.id}
                    className={classnames(props.classes.childItem, {
                        [props.classes.active]: item.active,
                        [props.classes.itemWeightLight]: (item.data.weight === TreeItemWeight.LIGHT && !item.active),
                        [props.classes.itemWeightDark]: (item.data.weight === TreeItemWeight.DARK && !item.active),
                    })}
                    style={{ paddingLeft: `${item.depth * props.levelIndentation}px` }}>
                    {isInFavoritesTree(props.it) ?
                     <div className={props.classes.indentSpacer} />
:
                     <i data-action={FLAT_TREE_ACTIONS.toggleOpen} className={props.classes.toggableIconContainer}>
                         <ListItemIcon className={props.getToggableIconClassNames(item.open, item.active)}>
                             {props.getProperArrowAnimation(item.status, item.items!)}
                         </ListItemIcon>
                     </i>}
                    {props.showSelection(item) && !props.useRadioButtons &&
                     <Checkbox
                         checked={item.selected}
                         className={props.classes.checkbox}
                         color="primary"
                         onClick={props.handleCheckboxChange(item)} />}
                    {props.showSelection(item) && props.useRadioButtons &&
                     <Radio
                         checked={item.selected}
                         className={props.classes.checkbox}
                         color="primary" />}
                    <div data-action={FLAT_TREE_ACTIONS.toggleActive} className={props.classes.renderContainer} ref={item.active ? props.selectedRef : undefined}>
                    <span className={props.classes.childLi}>
                    <ItemIcon type={item.data.type} active={item.active} kind={item.data.kind} headKind={item.data.headKind || null} groupClass={item.data.kind === ResourceKind.GROUP ? item.data.groupClass : ''} classes={props.classes} />
                    <span className={props.classes.childItemName}>
                        {item.data.name}
                    </span>
                    {
                        !!item.data.frozenByUuid ? <FreezeIcon className={props.classes.frozenIcon} /> : null
                    }
                        </span>
                    </div>
                </div>)
        }
    </div>;

function treePickerToTreeItems<T>(tree: Tree<T>, resources: ResourcesState){
    return function(id: string): TreeItem<any> {
        const node = getNode(id)(tree) || initTreeNode({ id: '', value: 'InvalidNode' });
        const items = getNodeChildrenIds(node.id)(tree)
            .map(treePickerToTreeItems(tree, resources));
        const resource = resources[node.id] as (Resource | undefined);

        return {
            active: node.active,
            data: resource
                ? {
                    ...resource,
                    name: typeof node.value === "string"
                        ? node.value
                        : typeof (node.value as any).name === "string"
                        ? (node.value as any).name
                        : "",
                    weight: (node.value as any).weight
                }
                : node.value,
            id: node.id,
            items: items.length > 0 ? items : undefined,
            open: node.expanded,
            selected: node.selected,
            status: TreeItemStatus[node.status],
        };
    };
}
type ItemsMap<T> = Map<string, TreeItem<T>>;

function flatTree<T>(itemsMap: ItemsMap<T>, depth: number, items?: TreeItem<T>[]): TreeItem<T>[]{
    return items ? items
        .map((item) => addToItemsMap(item, itemsMap))
        .reduce((acc, next) => {
            const { items } = next;
            acc.push({ ...next, depth });
            acc.push(...(next.open ? flatTree(itemsMap, depth + 1, items) : []));
            return acc;
        }, [] as TreeItem<T>[]) : [];
};

function addToItemsMap<T>(item: TreeItem<T>, itemsMap: Map<string, TreeItem<T>>): TreeItem<T> {
    itemsMap[item.id] = item;
    return item;
};

export const TreeComponent = withStyles(styles)(
    React.memo(function<T>(props: TreeProps<T> & WithStyles<CssRules>) {
        const level = props.level ? props.level : 0;
        const { classes, render, toggleItemActive, toggleItemOpen, currentItemUuid, useRadioButtons, resources, treePicker, pickerId } = props;
        const pickedTree = treePicker && pickerId ? treePicker[pickerId] : createTree<T>();
        const tree = props.tree || pickedTree;
        const { list, listItem, loader, toggableIconContainer, renderContainer } = classes;
        const itemsMap: ItemsMap<T> = new Map();
        const fillMap = (tree: Tree<T>, resources: ResourcesState) => getNodeChildrenIds('')(tree)
            .map(treePickerToTreeItems(tree, resources))
            .map(item => addToItemsMap<T>(item, itemsMap))
            .map(parentItem => ({
                ...parentItem,
                flatTree: true,
                items: flatTree(itemsMap, 2, parentItem.items || []),
            }))
        const items = tree && resources ? fillMap(tree, resources) : props.items;

        const showSelection = typeof props.showSelection === 'function'
            ? props.showSelection
            : () => props.showSelection ? true : false;

        const getProperArrowAnimation = (status: string, items: Array<TreeItem<T>>) => {
            return isSidePanelIconNotNeeded(status, items) ? <span /> : <SidePanelRightArrowIcon style={{ fontSize: '14px' }} data-cy="side-panel-arrow-icon" />;
        }

        const isSidePanelIconNotNeeded = (status: string, items: Array<TreeItem<T>>) => {
            return status === TreeItemStatus.PENDING ||
                (status === TreeItemStatus.LOADED && !items) ||
                (status === TreeItemStatus.LOADED && items && items.length === 0);
        }

        const getToggableIconClassNames = (isOpen?: boolean, isActive?: boolean) => {
            const { iconOpen, iconClose, active, toggableIcon } = props.classes;
            return classnames(toggableIcon, {
                [iconOpen]: isOpen,
                [iconClose]: !isOpen,
                [active]: isActive
            });
        }

        const handleCheckboxChange = (item: TreeItem<T>) => {
            const { toggleItemSelection } = props;
            return toggleItemSelection
                ? (event: React.MouseEvent<HTMLElement>) => {
                    event.stopPropagation();
                    toggleItemSelection(event, item);
                }
                : undefined;
        }

        const handleToggleItemOpen = (item: TreeItem<T>, event: React.MouseEvent<HTMLElement>) => {
            event.stopPropagation();
            props.toggleItemOpen(event, item);
        }

        // Scroll to selected item whenever it changes, accepts selectedRef from props for recursive trees
        const [cachedSelectedRef, setCachedRef] = useState<HTMLDivElement | null>(null)
        const scrollToNode = useCallback((node: HTMLDivElement | null) => {
            if (node && node.scrollIntoView && node !== cachedSelectedRef) {
                node.scrollIntoView({ behavior: "smooth", block: "center" });
            }
            setCachedRef(node);
        }, [cachedSelectedRef])
        const selectedRef = props.selectedRef || scrollToNode;

        const { levelIndentation = 20, itemRightPadding = 20 } = props;
        return <List className={list}>
            {items && items.map((it: TreeItem<T>, idx: number) => {
                if (isInFavoritesTree(it) && it.open === true && it.items && it.items.length) {
                    it = { ...it, items: it.items.filter(item => item.depth && item.depth < 3) }
                }
                return <div data-cy="tree-top-level-item" key={`item/${level}/${it.id}`}>
                    <ListItem button className={listItem}
                        data-cy="tree-li"
                        style={{
                            paddingLeft: (level + 1) * levelIndentation,
                            paddingRight: itemRightPadding,
                        }}
                        disableRipple={true}
                        onClick={event => toggleItemActive(event, it)}
                        selected={showSelection(it) && it.id === currentItemUuid}
                        onContextMenu={(event) => props.onContextMenu(event, it)}>
                        {it.status === TreeItemStatus.PENDING ?
                            <CircularProgress size={10} className={loader} /> : null}
                        <i onClick={(e) => handleToggleItemOpen(it, e)}
                            className={toggableIconContainer}>
                            <ListItemIcon className={getToggableIconClassNames(it.open, it.active)}
                                data-cy={`tree-item-toggle-${kebabCase(it.id.toString())}`}
                                >
                                {getProperArrowAnimation(it.status, it.items!)}
                            </ListItemIcon>
                        </i>
                        {showSelection(it) && !useRadioButtons &&
                            <Checkbox
                                checked={it.selected}
                                indeterminate={!it.selected && it.indeterminate}
                                className={classes.checkbox}
                                color="primary"
                                onClick={handleCheckboxChange(it)} />}
                        {showSelection(it) && useRadioButtons &&
                            <Radio
                                checked={it.selected}
                                className={classes.checkbox}
                                color="primary" />}
                        <div className={renderContainer} ref={!!it.active ? selectedRef : undefined}>
                            {render(it, level)}
                        </div>
                    </ListItem>
                    {
                        it.open && it.items && it.items.length > 0 &&
                            it.flatTree ?
                            <FlatTree
                                it={it}
                                itemsMap={itemsMap}
                                showSelection={showSelection}
                                classes={props.classes}
                                useRadioButtons={useRadioButtons}
                                levelIndentation={levelIndentation}
                                handleCheckboxChange={handleCheckboxChange}
                                onContextMenu={props.onContextMenu}
                                handleToggleItemOpen={handleToggleItemOpen}
                                toggleItemActive={props.toggleItemActive}
                                getToggableIconClassNames={getToggableIconClassNames}
                                getProperArrowAnimation={getProperArrowAnimation}
                                selectedRef={selectedRef}
                            /> :
                            <Collapse in={it.open} timeout="auto" unmountOnExit>
                                <TreeComponent
                                    tree={props.tree}
                                    resources={props.resources}
                                    showSelection={props.showSelection}
                                    items={it.items}
                                    render={render}
                                    toggleItemOpen={toggleItemOpen}
                                    toggleItemActive={toggleItemActive}
                                    level={level + 1}
                                    onContextMenu={props.onContextMenu}
                                    toggleItemSelection={props.toggleItemSelection}
                                    selectedRef={selectedRef}
                                />
                            </Collapse>
                    }
                </div>;
            })}
        </List>;
    }, preventRerender)
);

// return true to prevent re-render, false to allow re-render
function preventRerender(prevProps: TreeProps<any>, nextProps: TreeProps<any>) {
    if (prevProps.treePicker && nextProps.treePicker && prevProps.pickerId && nextProps.pickerId) {
        if (!isEqual(prevProps.treePicker[prevProps.pickerId], nextProps.treePicker[nextProps.pickerId])) {
            return false;
        }
    }
    return true;
}
