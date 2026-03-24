// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ProjectIcon } from "components/icon/icon";
import { PROJECT_PANEL_RUN_ID } from "store/project-panel/project-panel-action-bind";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { ProjectResource } from 'models/project';
import { connect } from "react-redux";
import { RootState } from "store/store";
import { getProjectPanelCurrentUuid } from "store/project-panel/project-panel";
import { getResource } from "store/resources/resources";

const DEFAULT_VIEW_MESSAGES = ['No workflow runs found'];

interface ProjectPanelRunProps {
    project?: ProjectResource;
    paperClassName?: string;
    onRowClick: (uuid: string) => void;
    onRowDoubleClick: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => void;
}

const mapStateToProps = (state: RootState): Pick<ProjectPanelRunProps, 'project'> => {
    const projectUuid = getProjectPanelCurrentUuid(state);
    const project = getResource<ProjectResource>(projectUuid)(state.resources);
    return {
        project,
    };
};

export const ProjectPanelRun = connect(mapStateToProps)((props: ProjectPanelRunProps) => (
    <DataExplorer
        id={PROJECT_PANEL_RUN_ID}
        onRowClick={props.onRowClick}
        onRowDoubleClick={props.onRowDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={false}
        defaultViewIcon={ProjectIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        parentResource={props.project}
        paperClassName={props.paperClassName}
    />
));
