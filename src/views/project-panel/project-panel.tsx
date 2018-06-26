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
import ProjectExplorer from "../../views-components/project-explorer/project-explorer";
import { projectExplorerItems } from "./project-panel-selectors";
import { ProjectExplorerItem } from "../../views-components/project-explorer/project-explorer-item";
import { Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

interface ProjectPanelDataProps {
    projects: ProjectState;
    collections: CollectionState;
}

type ProjectPanelProps = ProjectPanelDataProps & RouteComponentProps<{ name: string }> & DispatchProp;

class ProjectPanel extends React.Component<ProjectPanelProps & WithStyles<CssRules>> {
    render() {
        const items = projectExplorerItems(
            this.props.projects.items,
            this.props.projects.currentItemId,
            this.props.collections
        );
        return (
            <div>
                <div className={this.props.classes.toolbar}>
                    <Button color="primary" variant="raised" className={this.props.classes.button}>
                        Create a collection
                    </Button>
                    <Button color="primary" variant="raised" className={this.props.classes.button}>
                        Run a process
                    </Button>
                    <Button color="primary" variant="raised" className={this.props.classes.button}>
                        Create a project
                    </Button>
                </div>
                <ProjectExplorer
                    items={items}
                    onRowClick={this.goToItem}
                />
            </div>
        );
    }

    goToItem = (item: ProjectExplorerItem) => {
        this.props.dispatch<any>(setProjectItem(this.props.projects.items, item.uuid, item.kind, ItemMode.BOTH));
    }
}

type CssRules = "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = theme => ({
    toolbar: {
        marginBottom: theme.spacing.unit * 3,
        display: "flex",
        justifyContent: "flex-end"
    },
    button: {
        marginLeft: theme.spacing.unit
    }
});

export default withStyles(styles)(
    connect(
        (state: RootState) => ({
            projects: state.projects,
            collections: state.collections
        })
    )(ProjectPanel));
