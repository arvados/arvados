// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ReactElement } from 'react';
import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import ListItemText from "@material-ui/core/ListItemText/ListItemText";
import ListItemIcon from '@material-ui/core/ListItemIcon';
import Typography from '@material-ui/core/Typography';

import Tree, { TreeItem } from '../tree/tree';
import { Project } from '../../models/project';

type CssRules = 'active' | 'row' | 'treeContainer';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    active: {
        color: '#4285F6',
    },
    row: {
        display: 'flex',
        alignItems: 'center',
        marginLeft: '20px',
    },
    treeContainer: {
        position: 'absolute',
        overflowX: 'visible',
        marginTop: '80px',
        minWidth: '240px',
        whiteSpace: 'nowrap',
    }
});

export interface WorkbenchProps {
    projects: Array<TreeItem<Project>>;
    toggleProjectTreeItem: (id: string) => any;
}

class ProjectTree<T> extends React.Component<WorkbenchProps & WithStyles<CssRules>> {
    render(): ReactElement<any> {
        const {classes, projects} = this.props;
        return (
            <div className={classes.treeContainer}>
                <Tree items={projects}
                    toggleItem={this.props.toggleProjectTreeItem}
                    render={(project: TreeItem<Project>) => <span className={classes.row}>
                        <div><ListItemIcon className={project.active ? classes.active : ''}>{project.data.icon}</ListItemIcon></div>
                        <div><ListItemText primary={<Typography className={project.active ? classes.active : ''}>{project.data.name}</Typography>} /></div>
                    </span>} />
            </div>
        );
    }
}

export default withStyles(styles)(ProjectTree)