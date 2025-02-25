// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { TreePicker, TreePickerProps } from "../tree-picker/tree-picker";
import { TreeItem } from "components/tree/tree";
import { ProjectResource } from "models/project";
import { ListItemTextIcon } from "components/list-item-text-icon/list-item-text-icon";
import { activateSidePanelTreeItem,
         toggleSidePanelTreeItemCollapse,
         SIDE_PANEL_TREE,
         SidePanelTreeCategory,
         getSidePanelIcon
} from 'store/side-panel-tree/side-panel-tree-actions';
import { openSidePanelContextMenu } from 'store/context-menu/context-menu-actions';
import { noop } from 'lodash';
import { ResourceKind } from "models/resource";
import { IllegalNamingWarning } from "components/warning/warning";
import { GroupClass } from "models/group";
import { setSelectedResourceUuid } from "store/selected-resource/selected-resource-actions";
import { FilterGroupIcon, ProjectsIcon } from 'components/icon/icon';

export interface SidePanelTreeProps {
    onItemActivation: (id: string) => void;
    sidePanelProgress?: boolean;
    isCollapsed?: boolean;
    currentRoute?: string;
}

type SidePanelTreeActionProps = Pick<TreePickerProps<ProjectResource | string>, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen' | 'toggleItemSelection'>;

const mapDispatchToProps = (dispatch: Dispatch, props: SidePanelTreeProps): SidePanelTreeActionProps => ({
    onContextMenu: (event, { id }) => {
        dispatch<any>(openSidePanelContextMenu(event, id));
    },
    toggleItemActive: (_, { id }) => {
        dispatch<any>(activateSidePanelTreeItem(id));
        const isSidePanelCat = Object.values(SidePanelTreeCategory).includes(id as SidePanelTreeCategory);
        dispatch<any>(setSelectedResourceUuid(isSidePanelCat ? null : id));
        props.onItemActivation(id);
    },
    toggleItemOpen: (_, { id }) => {
        dispatch<any>(toggleSidePanelTreeItemCollapse(id));
    },
    toggleItemSelection: noop,
});

export const SidePanelTree = connect(undefined, mapDispatchToProps)(
    (props: SidePanelTreeActionProps) =>
        <div data-cy="side-panel-tree">
            <TreePicker {...props} render={renderSidePanelItem} pickerId={SIDE_PANEL_TREE} />
        </div>);

const renderSidePanelItem = (item: TreeItem<ProjectResource>) => {
    const name = typeof item.data === 'string' ? item.data : item.data.name;
    const warn = typeof item.data !== 'string' && item.data.kind === ResourceKind.PROJECT
        ? <IllegalNamingWarning name={name} />
        : undefined;
    return <ListItemTextIcon
        icon={getProjectPickerIcon(item)}
        name={name}
        nameDecorator={warn}
        isActive={item.active}
        hasMargin={true}
    />;
};

const getProjectPickerIcon = (item: TreeItem<ProjectResource | string>) =>
    typeof item.data === 'string'
        ? getSidePanelIcon(item.data)
        : (item.data && item.data.groupClass === GroupClass.FILTER)
            ? FilterGroupIcon
            : ProjectsIcon;
