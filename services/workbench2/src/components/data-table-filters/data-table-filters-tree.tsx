// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { toggleNodeSelection, getNode, initTreeNode, getNodeChildrenIds, selectNode, deselectNodes } from 'models/tree';
import { TreeComponent, TreeItem, TreeItemStatus} from 'components/tree/tree';
import { noop, map } from "lodash/fp";
import { toggleNodeCollapse } from 'models/tree';
import { countNodes, countChildren } from 'models/tree';
import { DataTableFilterItem, DataTableFilters } from './data-table-filters';

export type ColumnFilterCount = Record<string, number>;
export type ColumnFilterCounts = Record<string, ColumnFilterCount>;

export interface DataTableFilterProps {
    filters: DataTableFilters;
    onChange?: (filters: DataTableFilters) => void;

    /**
     * When set to true, only one filter can be selected at a time.
     */
    mutuallyExclusive?: boolean;
    columnFilterCount: ColumnFilterCount;
}

export class DataTableFiltersTree extends React.Component<DataTableFilterProps> {

    render() {
        const { filters, columnFilterCount } = this.props;
        const hasSubfilters = countNodes(filters) !== countChildren('')(filters);
        return <TreeComponent
            key={JSON.stringify(columnFilterCount)}
            levelIndentation={hasSubfilters ? 20 : 0}
            itemRightPadding={20}
            items={filtersToTree(filters, columnFilterCount)}
            render={this.props.mutuallyExclusive ? renderRadioItem : renderItem}
            showSelection
            useRadioButtons={this.props.mutuallyExclusive}
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
        const filters = selectNode(item.id, true)(this.props.filters);
        const toDeselect = Object.keys(this.props.filters).filter((id) => (id !== item.id));
        onChange(deselectNodes(toDeselect, true)(filters));
    }

    toggleFilter = (_: React.MouseEvent, item: TreeItem<DataTableFilterItem>) => {
        const { onChange = noop } = this.props;
        onChange(toggleNodeSelection(item.id, true)(this.props.filters));
    }

    toggleOpen = (_: React.MouseEvent, item: TreeItem<DataTableFilterItem>) => {
        const { onChange = noop } = this.props;
        onChange(toggleNodeCollapse(item.id)(this.props.filters));
    }
}

const renderedItemStyles = {
    root: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        '&hover': {
            color: 'grey',
        },
    },
    name: {
        marginRight: '10px',
    },
};

const renderItem = ({data: {name, count}, initialState, selected}: TreeItem<DataTableFilterItem>) =>
    count ? <div style={renderedItemStyles.root}>
                <span style={renderedItemStyles.name}>{name}</span>
                <span>{`${count.toLocaleString() || '0'}`}</span>
                {initialState !== selected ? <>
                    *
                </> : null}
            </div>
            :
            <span>
                {name}{initialState !== selected ? <>*</> : null}
            </span>;

const renderRadioItem = ({data: {name, count}}: TreeItem<DataTableFilterItem>) =>
    <div style={renderedItemStyles.root}>
        <span style={renderedItemStyles.name}>{name}</span>
        <span>{`${count?.toLocaleString() || '0'}`}</span>
    </div>;

const filterToTreeItem = (filters: DataTableFilters, columnFilterCount: ColumnFilterCount) =>
    (id: string): TreeItem<any> => {
        const filterValue = filters[id].value;
        if (filterValue) {
            filterValue['count'] = columnFilterCount[id]
        }
        const node = getNode(id)(filters) || initTreeNode({ id: '', value: 'InvalidNode' });
        const items = getNodeChildrenIds(node.id)(filters)
            .map(filterToTreeItem(filters, columnFilterCount));
        const isIndeterminate = !node.selected && items.some(i => i.selected || i.indeterminate);

        return {
            active: node.active,
            data: node.value,
            id: node.id,
            items: items.length > 0 ? items : undefined,
            open: node.expanded,
            selected: node.selected,
            initialState: node.initialState,
            indeterminate: isIndeterminate,
            status: TreeItemStatus.LOADED,
        };
    };

const filtersToTree = (filters: DataTableFilters, columnFilterCount: ColumnFilterCount): TreeItem<DataTableFilterItem>[] =>
    map(filterToTreeItem(filters, columnFilterCount), getNodeChildrenIds('')(filters));
