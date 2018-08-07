// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { Typography } from "@material-ui/core";
import { TreePicker } from "../tree-picker/tree-picker";
import { TreeProps, TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { ProjectResource } from "../../models/project";
import { treePickerActions } from "../../store/tree-picker/tree-picker-actions";
import { ListItemTextIcon } from "../../components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon } from "../../components/icon/icon";
import { createTreePickerNode } from "../../store/tree-picker/tree-picker";
import { RootState } from "../../store/store";
import { ServiceRepository } from "../../services/services";
import { FilterBuilder } from "../../common/api/filter-builder";

type ProjectTreePickerProps = Pick<TreeProps<ProjectResource>, 'toggleItemActive' | 'toggleItemOpen'>;

const mapDispatchToProps = (dispatch: Dispatch, props: {onChange: (projectUuid: string) => void}): ProjectTreePickerProps => ({
    toggleItemActive: id => {
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ id }));
        props.onChange(id);
    },
    toggleItemOpen: (id, status) => {
        status === TreeItemStatus.INITIAL
            ? dispatch<any>(loadProjectTreePickerProjects(id))
            : dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id }));
    }
});

export const ProjectTreePicker = connect(undefined, mapDispatchToProps)((props: ProjectTreePickerProps) =>
    <div style={{display: 'flex', flexDirection: 'column'}}>
        <Typography variant='caption' style={{flexShrink: 0}}>
            Select a project
        </Typography>
        <div style={{flexGrow: 1, overflow: 'auto'}}>
            <TreePicker {...props} render={renderTreeItem} />
        </div>
    </div>);

// TODO: move action creator to store directory
export const loadProjectTreePickerProjects = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id }));

        const ownerUuid = id.length === 0 ? services.authService.getUuid() || '' : id;

        const filters = FilterBuilder
            .create<ProjectResource>()
            .addEqual('ownerUuid', ownerUuid);

        const { items } = await services.projectService.list({ filters });

        dispatch<any>(receiveProjectTreePickerData(id, items));
    };

const renderTreeItem = (item: TreeItem<ProjectResource>) =>
    <ListItemTextIcon
        icon={ProjectIcon}
        name={item.data.name}
        isActive={item.active}
        hasMargin={true} />;

// TODO: move action creator to store directory
const receiveProjectTreePickerData = (id: string, projects: ProjectResource[]) =>
    (dispatch: Dispatch) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id,
            nodes: projects.map(project => createTreePickerNode({ id: project.uuid, value: project }))
        }));
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id }));
    };