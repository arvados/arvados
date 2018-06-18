// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import ListItemText from "@material-ui/core/ListItemText/ListItemText";
import ListItemIcon from '@material-ui/core/ListItemIcon';
import Typography from '@material-ui/core/Typography';

import Tree, { TreeItem, TreeItemStatus } from '../../components/tree/tree';
import { Project } from '../../models/project';

type CssRules = 'active' | 'listItemText' | 'row' | 'treeContainer';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    active: {
        color: '#4285F6',
    },
    listItemText: {
        padding: '0px',
    },
    row: {
        display: 'flex',
        alignItems: 'center',
        marginLeft: '20px',
    },
    treeContainer: {
        marginTop: '37px',
        overflowX: 'visible',
        overflowY: 'auto',
        minWidth: '240px',
        whiteSpace: 'nowrap',
    }
});

export interface ProjectTreeProps {
    projects: Array<TreeItem<Project>>;
    toggleProjectTreeItem: (id: string, status: TreeItemStatus) => void;
}

class ProjectTree<T> extends React.Component<ProjectTreeProps & WithStyles<CssRules>> {
    render(): ReactElement<any> {
        const {classes, projects} = this.props;
        const {active, listItemText, row, treeContainer} = classes;
        return (
            <div className={treeContainer}>
                <Tree items={projects}
                    toggleItem={this.props.toggleProjectTreeItem}
                    render={(project: TreeItem<Project>, level: number) =>
                        <span className={row}>
                            <ListItemIcon className={project.active ? active : ''}>
                                {level === 0 ? <i className="fas fa-th"/> : <i className="fas fa-folder"/>}
                            </ListItemIcon>
                            <ListItemText className={listItemText} primary={
                                <Typography className={project.active ? active : ''}>
                                    {project.data.name}
                                </Typography>
                            }/>
                        </span>
                    }/>
            </div>
        );
    }
}

export default withStyles(styles)(ProjectTree);
