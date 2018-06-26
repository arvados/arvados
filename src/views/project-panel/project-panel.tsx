// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { RouteComponentProps } from 'react-router';
import { ProjectState } from '../../store/project/project-reducer';
import { RootState } from '../../store/store';
import { connect, DispatchProp } from 'react-redux';
import { CollectionState } from "../../store/collection/collection-reducer";
import { ItemMode, setProjectItem } from "../../store/navigation/navigation-action";
import ProjectExplorer, { ProjectExplorerContextActions } from "../../views-components/project-explorer/project-explorer";
import { projectExplorerItems } from "./project-panel-selectors";
import { ProjectExplorerItem } from "../../views-components/project-explorer/project-explorer-item";

interface ProjectPanelDataProps {
    projects: ProjectState;
    collections: CollectionState;
}

type ProjectPanelProps = ProjectPanelDataProps & RouteComponentProps<{ name: string }> & DispatchProp;

class ProjectPanel extends React.Component<ProjectPanelProps> {
    render() {
        const items = projectExplorerItems(
            this.props.projects.items,
            this.props.projects.currentItemId,
            this.props.collections
        );
        return (
            <ProjectExplorer
                items={items}
                onRowClick={this.goToItem}
            />
        );
    }

    goToItem = (item: ProjectExplorerItem) => {
        this.props.dispatch<any>(setProjectItem(this.props.projects.items, item.uuid, item.kind, ItemMode.BOTH));
    }
}

export default connect(
    (state: RootState) => ({
        projects: state.projects,
        collections: state.collections
    })
)(ProjectPanel);
