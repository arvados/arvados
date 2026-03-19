// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ProjectIcon } from "components/icon/icon";
import { PROJECT_PANEL_DATA_ID } from "store/project-panel/project-panel-action-bind";
import { DataExplorer } from "views-components/data-explorer/data-explorer";

const DEFAULT_VIEW_MESSAGES = ['No data found'];

interface ProjectPanelDataProps {
    paperClassName?: string;
    onRowClick: (uuid: string) => void;
    onRowDoubleClick: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => void;
};

export const ProjectPanelData = (props: ProjectPanelDataProps) => (
    <DataExplorer
        id={PROJECT_PANEL_DATA_ID}
        onRowClick={props.onRowClick}
        onRowDoubleClick={props.onRowDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={false}
        defaultViewIcon={ProjectIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        paperClassName={props.paperClassName}
    />
);
