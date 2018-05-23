// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';

import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import { connect } from "react-redux";
import Tree from "../../components/tree/tree";
import { Project } from "../../models/project";
import { RootState } from "../../store/root-reducer";

const drawerWidth = 240;

type CssRules = 'root' | 'appBar' | 'drawerPaper' | 'content' | 'toolbar';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        flexGrow: 1,
        zIndex: 1,
        overflow: 'hidden',
        position: 'relative',
        display: 'flex',
        width: '100%',
        height: '100%'
    },
    appBar: {
        zIndex: theme.zIndex.drawer + 1,
        backgroundColor: '#692498'
    },
    drawerPaper: {
        position: 'relative',
        width: drawerWidth,
    },
    content: {
        flexGrow: 1,
        backgroundColor: theme.palette.background.default,
        padding: theme.spacing.unit * 3,
        minWidth: 0,
    },
    toolbar: theme.mixins.toolbar
});

interface WorkbenchProps {
    projects: Project[]
}

interface WorkbenchState {
}

class Workbench extends React.Component<WorkbenchProps & WithStyles<CssRules>, WorkbenchState> {
    render() {
        const {classes} = this.props;

        return (
            <div className={classes.root}>
                <AppBar position="absolute" className={classes.appBar}>
                    <Toolbar>
                        <Typography variant="title" color="inherit" noWrap>
                            Arvados<br/>Workbench 2
                        </Typography>
                    </Toolbar>
                </AppBar>
                <Drawer
                    variant="permanent"
                    classes={{
                        paper: classes.drawerPaper,
                    }}>
                    <div className={classes.toolbar}/>
                    <Tree items={this.props.projects} render={(p: Project) =>
                        <span>{p.name}</span>
                    }/>
                </Drawer>
                <main className={classes.content}>
                    <div className={classes.toolbar}/>
                    <Typography noWrap>Hello new workbench!</Typography>
                </main>
            </div>
        );
    }
}

export default connect<WorkbenchProps>(
    (state: RootState) => ({
        projects: state.projects
    })
)(
    withStyles(styles)(Workbench)
);
