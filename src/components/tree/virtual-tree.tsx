// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import { FixedSizeList, ListChildComponentProps } from "react-window";
import AutoSizer from "react-virtualized-auto-sizer";
// import {FixedSizeTree as Tree} from 'react-vtree';

import { ArvadosTheme } from '~/common/custom-theme';
import { TreeItem } from './tree';
// import { FileTreeData } from '../file-tree/file-tree-data';

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

// export const RowA = <T, _>(items: TreeItem<T>[], render:any) => (index: number) => {
//     return <div>
//         {render(items[index])}
//     </div>;
// };

// For some reason, on TSX files it isn't accepted just one generic param, so
// I'm using <T, _> as a workaround.
export const Row = <T, _>(items: TreeItem<T>[], render: any) => (props: React.PropsWithChildren<ListChildComponentProps>) => {
    const { index, style } = props;
    const level = items[index].level || 0;
    const levelIndentation = 20;
    return <div style={style}>
        <div style={{ paddingLeft: (level + 1) * levelIndentation,}}>
            {typeof render === 'function'
                ? items[index] && render(items[index]) || ''
                : 'whoops'}
        </div>
    </div>;
    // <div style={style} key={`item/${level}/${idx}`}>
    //     <ListItem button className={listItem}
    //         style={{
    //             paddingLeft: (level + 1) * levelIndentation,
    //             paddingRight: itemRightPadding,
    //         }}
    //         disableRipple={disableRipple}
    //         onClick={event => toggleItemActive(event, it)}
    //         selected={showSelection(it) && it.id === currentItemUuid}
    //         onContextMenu={this.handleRowContextMenu(it)}>
    //         {it.status === TreeItemStatus.PENDING ?
    //             <CircularProgress size={10} className={loader} /> : null}
    //         <i onClick={this.handleToggleItemOpen(it)}
    //             className={toggableIconContainer}>
    //             <ListItemIcon className={this.getToggableIconClassNames(it.open, it.active)}>
    //                 {this.getProperArrowAnimation(it.status, it.items!)}
    //             </ListItemIcon>
    //         </i>
    //         {showSelection(it) && !useRadioButtons &&
    //             <Checkbox
    //                 checked={it.selected}
    //                 className={classes.checkbox}
    //                 color="primary"
    //                 onClick={this.handleCheckboxChange(it)} />}
    //         {showSelection(it) && useRadioButtons &&
    //             <Radio
    //                 checked={it.selected}
    //                 className={classes.checkbox}
    //                 color="primary" />}
    //         <div className={renderContainer}>
    //             {render(it, level)}
    //         </div>
    //     </ListItem>
    //     {it.items && it.items.length > 0 &&
    //         <Collapse in={it.open} timeout="auto" unmountOnExit>
    //             <Tree
    //                 showSelection={this.props.showSelection}
    //                 items={it.items}
    //                 render={render}
    //                 disableRipple={disableRipple}
    //                 toggleItemOpen={toggleItemOpen}
    //                 toggleItemActive={toggleItemActive}
    //                 level={level + 1}
    //                 onContextMenu={onContextMenu}
    //                 toggleItemSelection={this.props.toggleItemSelection} />
    //         </Collapse>}
    // </div>
};

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

// const treeWalkerWithTree = (tree: Array<TreeItem<FileTreeData>>) => function* treeWalker(refresh: any) {
//     const stack = [];

//     // Remember all the necessary data of the first node in the stack.
//     stack.push({
//       nestingLevel: 0,
//       node: tree,
//     });

//     // Walk through the tree until we have no nodes available.
//     while (stack.length !== 0) {
//         const {
//             node: {items = [], id, name},
//             nestingLevel,
//         } = stack.pop()!;

//         // Here we are sending the information about the node to the Tree component
//         // and receive an information about the openness state from it. The
//         // `refresh` parameter tells us if the full update of the tree is requested;
//         // basing on it we decide to return the full node data or only the node
//         // id to update the nodes order.
//         const isOpened = yield refresh
//             ? {
//                 id,
//                 isLeaf: items.length === 0,
//                 isOpenByDefault: true,
//                 name,
//                 nestingLevel,
//             }
//             : id;

//         // Basing on the node openness state we are deciding if we need to render
//         // the child nodes (if they exist).
//         if (children.length !== 0 && isOpened) {
//             // Since it is a stack structure, we need to put nodes we want to render
//             // first to the end of the stack.
//             for (let i = children.length - 1; i >= 0; i--) {
//                 stack.push({
//                     nestingLevel: nestingLevel + 1,
//                     node: children[i],
//                 });
//             }
//         }
//     }
// };

// // Node component receives all the data we created in the `treeWalker` +
// // internal openness state (`isOpen`), function to change internal openness
// // state (`toggle`) and `style` parameter that should be added to the root div.
// const Node = ({data: {isLeaf, name}, isOpen, style, toggle}) => (
//     <div style={style}>
//         {!isLeaf && (
//         <button type="button" onClick={toggle}>
//             {isOpen ? '-' : '+'}
//         </button>
//         )}
//         <div>{name}</div>
//     </div>
// );

// export const Example = () => (
//     <Tree treeWalker={treeWalker} itemSize={30} height={150} width={300}>
//         {Node}
//     </Tree>
// );