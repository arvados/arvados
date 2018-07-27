// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Tree, TreeItem, TreeItemStatus } from "../tree/tree";
import { FileTreeData } from "./file-tree-data";
import { FileTreeItem } from "./file-tree-item";

export interface FileTreeProps {
    items: Array<TreeItem<FileTreeData>>;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
}

export class FileTree extends React.Component<FileTreeProps> {
    render() {
        return <Tree
            showSelection={true}
            items={this.props.items}
            disableRipple={true}
            render={this.renderItem}
            onContextMenu={this.handleContextMenu}
            toggleItemActive={this.handleToggleActive}
            toggleItemOpen={this.handleToggle}
            onSelectionChange={this.handleSelectionChange} />;
    }

    handleContextMenu = (event: React.MouseEvent<any>, item: TreeItem<FileTreeData>) => {
        event.stopPropagation();
        this.props.onContextMenu(event, item);
    }

    handleToggle = (id: string, status: TreeItemStatus) => {
        this.props.onCollapseToggle(id, status);
    }

    handleToggleActive = () => { return; };

    handleSelectionChange = (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => {
        event.stopPropagation();
        this.props.onSelectionToggle(event, item);
    }

    renderItem = (item: TreeItem<FileTreeData>) =>
        <FileTreeItem
            item={item}
            onMoreClick={this.handleContextMenu} />

}
