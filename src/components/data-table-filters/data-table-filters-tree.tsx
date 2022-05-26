// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Tree, toggleNodeSelection, getNode, initTreeNode, getNodeChildrenIds, selectNode, deselectNodes } from 'models/tree';
import { Tree as TreeComponent, TreeItem, TreeItemStatus } from 'components/tree/tree';
import { noop, map } from "lodash/fp";
import { toggleNodeCollapse } from 'models/tree';
import { countNodes, countChildren } from 'models/tree';

export interface DataTableFilterItem {
    name: string;
}

export type DataTableFilters = Tree<DataTableFilterItem>;

export interface DataTableFilterProps {
    filters: DataTableFilters;
    onChange?: (filters: DataTableFilters) => void;

    /**
     * When set to true, only one filter can be selected at a time.
     */
    mutuallyExclusive?: boolean;
}

export class DataTableFiltersTree extends React.Component<DataTableFilterProps> {

    render() {
        const { filters } = this.props;
        const hasSubfilters = countNodes(filters) !== countChildren('')(filters);
        return <TreeComponent
            levelIndentation={hasSubfilters ? 20 : 0}
            itemRightPadding={20}
            items={filtersToTree(filters)}
            render={renderItem}
            showSelection
            useRadioButtons={this.props.mutuallyExclusive}
            disableRipple
            onContextMenu={noop}
            toggleItemActive={
                this.props.mutuallyExclusive
                    ? this.toggleRadioButtonFilter
                    : this.toggleFilter
            }
            toggleItemOpen={this.toggleOpen}
        />;
    }

    /**
     * Handler for when a tree item is toggled via a radio button.
     * Ensures mutual exclusivity among filter tree items.
     */
    toggleRadioButtonFilter = (_: any, item: TreeItem<DataTableFilterItem>) => {
        const { onChange = noop } = this.props;

        // If the filter is already selected, do nothing.
        if (item.selected) { return; }

        // Otherwise select this node and deselect the others
        const filters = selectNode(item.id)(this.props.filters);
        const toDeselect = Object.keys(this.props.filters).filter((id) => (id !== item.id));
        onChange(deselectNodes(toDeselect)(filters));
    }

    toggleFilter = (_: React.MouseEvent, item: TreeItem<DataTableFilterItem>) => {
        const { onChange = noop } = this.props;
        onChange(toggleNodeSelection(item.id)(this.props.filters));
    }

    toggleOpen = (_: React.MouseEvent, item: TreeItem<DataTableFilterItem>) => {
        const { onChange = noop } = this.props;
        onChange(toggleNodeCollapse(item.id)(this.props.filters));
    }
}

const renderItem = (item: TreeItem<DataTableFilterItem>) =>
    <span>{item.data.name}</span>;

const filterToTreeItem = (filters: DataTableFilters) =>
    (id: string): TreeItem<any> => {
        const node = getNode(id)(filters) || initTreeNode({ id: '', value: 'InvalidNode' });
        const items = getNodeChildrenIds(node.id)(filters)
            .map(filterToTreeItem(filters));
        const isIndeterminate = !node.selected && items.some(i => i.selected || i.indeterminate);

        return {
            active: node.active,
            data: node.value,
            id: node.id,
            items: items.length > 0 ? items : undefined,
            open: node.expanded,
            selected: node.selected,
            indeterminate: isIndeterminate,
            status: TreeItemStatus.LOADED,
        };
    };

const filtersToTree = (filters: DataTableFilters): TreeItem<DataTableFilterItem>[] =>
    map(filterToTreeItem(filters), getNodeChildrenIds('')(filters));
