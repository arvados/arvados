// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';

import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import { connect, DispatchProp } from "react-redux";
import ProjectList from "../../components/project-list/project-list";
import { Route, Switch } from "react-router";
import { Link } from "react-router-dom";
import Button from "@material-ui/core/Button/Button";
import authActions from "../../store/auth/auth-action";
import IconButton from "@material-ui/core/IconButton/IconButton";
import Menu from "@material-ui/core/Menu/Menu";
import MenuItem from "@material-ui/core/MenuItem/MenuItem";
import { AccountCircle } from "@material-ui/icons";
import { User } from "../../models/user";
import Grid from "@material-ui/core/Grid/Grid";
import { RootState } from "../../store/store";
import projectActions from "../../store/project/project-action"

import ProjectTree from '../../components/project-tree/project-tree';
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { projectService } from '../../services/services';

const drawerWidth = 240;

type CssRules = 'root' | 'appBar' | 'drawerPaper' | 'content' | 'toolbar';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        flexGrow: 1,
        zIndex: 1,
        overflow: 'hidden',
        position: 'relative',
        display: 'flex',
        width: '100vw',
        height: '100vh'
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
        height: '100%',
        minWidth: 0,
    },
    toolbar: theme.mixins.toolbar
});

interface WorkbenchDataProps {
    projects: Array<TreeItem<Project>>;
    user?: User;
}

interface WorkbenchActionProps {
}

type WorkbenchProps = WorkbenchDataProps & WorkbenchActionProps & DispatchProp & WithStyles<CssRules>;

interface WorkbenchState {
    anchorEl: any;
}

class Workbench extends React.Component<WorkbenchProps, WorkbenchState> {
    constructor(props: WorkbenchProps) {
        super(props);
        this.state = {
            anchorEl: null
        }
    }

    login = () => {
        this.props.dispatch(authActions.LOGIN());
    };

    logout = () => {
        this.handleClose();
        this.props.dispatch(authActions.LOGOUT());
    };

    handleOpenMenu = (event: React.MouseEvent<any>) => {
        this.setState({
            anchorEl: event.currentTarget
        });
    };

    handleClose = () => {
        this.setState({
            anchorEl: null
        });
    };

    toggleProjectTreeItem = (itemId: string, status: TreeItemStatus) => {
        if (status === TreeItemStatus.Loaded) {
            this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(itemId))
        } else {
            this.props.dispatch<any>(projectService.getProjectList(itemId)).then(() => {
                this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(itemId));
            })
        }
    };

    render() {
        const { classes, user } = this.props;
        return (
            <div className={classes.root}>
                <AppBar position="absolute" className={classes.appBar}>
                    <Toolbar>
                        <Typography variant="title" color="inherit" noWrap style={{ flexGrow: 1 }}>
                            <span>Arvados</span><br /><span style={{ fontSize: 12 }}>Workbench 2</span>
                        </Typography>
                        {user ?
                            <Grid container style={{ width: 'auto' }}>
                                <Grid container style={{ width: 'auto' }} alignItems='center'>
                                    <Typography variant="title" color="inherit" noWrap>
                                        {user.firstName} {user.lastName}
                                    </Typography>
                                </Grid>
                                <Grid item>
                                    <IconButton
                                        aria-owns={this.state.anchorEl ? 'menu-appbar' : undefined}
                                        aria-haspopup="true"
                                        onClick={this.handleOpenMenu}
                                        color="inherit">
                                        <AccountCircle />
                                    </IconButton>
                                </Grid>
                                <Menu
                                    id="menu-appbar"
                                    anchorEl={this.state.anchorEl}
                                    anchorOrigin={{
                                        vertical: 'top',
                                        horizontal: 'right',
                                    }}
                                    transformOrigin={{
                                        vertical: 'top',
                                        horizontal: 'right',
                                    }}
                                    open={!!this.state.anchorEl}
                                    onClose={this.handleClose}>
                                    <MenuItem onClick={this.logout}>Logout</MenuItem>
                                    <MenuItem onClick={this.handleClose}>My account</MenuItem>
                                </Menu>
                            </Grid>
                            :
                            <Button color="inherit" onClick={this.login}>Login</Button>
                        }
                    </Toolbar>
                </AppBar>
                {user &&
                    <Drawer
                        variant="permanent"
                        classes={{
                            paper: classes.drawerPaper,
                        }}>
                        <div className={classes.toolbar} />
                        <ProjectTree
                            projects={this.props.projects}
                            toggleProjectTreeItem={this.toggleProjectTreeItem} />
                    </Drawer>}
                <main className={classes.content}>
                    <div className={classes.toolbar} />
                    <Switch>
                        <Route path="/project/:name" component={ProjectList} />
                    </Switch>
                </main>
            </div>
        );
    }
}

export default connect<WorkbenchDataProps>(
    (state: RootState) => ({
        projects: state.projects,
        user: state.auth.user
    })
)(
    withStyles(styles)(Workbench)
);
