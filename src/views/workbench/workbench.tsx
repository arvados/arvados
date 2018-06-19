// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';

import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch } from "react-router";
import authActions from "../../store/auth/auth-action";
import { User } from "../../models/user";
import { RootState } from "../../store/store";
import MainAppBar, { MainAppBarActionProps, MainAppBarMenuItem } from '../../components/main-app-bar/main-app-bar';
import { Breadcrumb } from '../../components/breadcrumbs/breadcrumbs';
import { push } from 'react-router-redux';
import projectActions from "../../store/project/project-action";
import ProjectTree from '../../components/project-tree/project-tree';
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { projectService } from '../../services/services';
import { getTreePath } from '../../store/project/project-reducer';
import DataExplorer from '../data-explorer/data-explorer';

const drawerWidth = 240;
const appBarHeight = 102;

type CssRules = 'root' | 'appBar' | 'drawerPaper' | 'content' | 'contentWrapper' | 'toolbar';

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
        backgroundColor: '#692498',
        position: "absolute",
        width: "100%"
    },
    drawerPaper: {
        position: 'relative',
        width: drawerWidth,
    },
    contentWrapper: {
        backgroundColor: theme.palette.background.default,
        display: "flex",
        flexGrow: 1,
        minWidth: 0,
        paddingTop: appBarHeight
    },
    content: {
        padding: theme.spacing.unit * 3,
        overflowY: "auto",
        flexGrow: 1
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

interface NavBreadcrumb extends Breadcrumb {
    itemId: string;
    status: TreeItemStatus;
}

interface NavMenuItem extends MainAppBarMenuItem {
    action: () => void;
}

interface WorkbenchState {
    anchorEl: any;
    breadcrumbs: NavBreadcrumb[];
    searchText: string;
    menuItems: {
        accountMenu: NavMenuItem[],
        helpMenu: NavMenuItem[],
        anonymousMenu: NavMenuItem[]
    };
}

class Workbench extends React.Component<WorkbenchProps, WorkbenchState> {
    state = {
        anchorEl: null,
        searchText: "",
        breadcrumbs: [],
        menuItems: {
            accountMenu: [
                {
                    label: "Logout",
                    action: () => this.props.dispatch(authActions.LOGOUT())
                },
                {
                    label: "My account",
                    action: () => this.props.dispatch(push("/my-account"))
                }
            ],
            helpMenu: [
                {
                    label: "Help",
                    action: () => this.props.dispatch(push("/help"))
                }
            ],
            anonymousMenu: [
                {
                    label: "Sign in",
                    action: () => this.props.dispatch(authActions.LOGIN())
                }
            ]
        }
    };


    mainAppBarActions: MainAppBarActionProps = {
        onBreadcrumbClick: ({ itemId, status }: NavBreadcrumb) => {
            this.toggleProjectTreeItem(itemId, status);
        },
        onSearch: searchText => {
            this.setState({ searchText });
            this.props.dispatch(push(`/search?q=${searchText}`));
        },
        onMenuItemClick: (menuItem: NavMenuItem) => menuItem.action()
    };

    toggleProjectTreeItem = (itemId: string, status: TreeItemStatus) => {
        if (status === TreeItemStatus.Loaded) {
            this.openProjectItem(itemId);
        } else {
            this.props.dispatch<any>(projectService.getProjectList(itemId)).then(() => this.openProjectItem(itemId));
        }
    }

    openProjectItem = (itemId: string) => {
        const branch = getTreePath(this.props.projects, itemId);
        this.setState({
            breadcrumbs: branch.map(item => ({
                label: item.data.name,
                itemId: item.data.uuid,
                status: item.status
            }))
        });
        this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM(itemId));
        this.props.dispatch(push(`/project/${itemId}`));
    }

    render() {
        const { classes, user } = this.props;
        return (
            <div className={classes.root}>
                <div className={classes.appBar}>
                    <MainAppBar
                        breadcrumbs={this.state.breadcrumbs}
                        searchText={this.state.searchText}
                        user={this.props.user}
                        menuItems={this.state.menuItems}
                        {...this.mainAppBarActions}
                    />
                </div>
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
                <main className={classes.contentWrapper}>
                    <div className={classes.content}>
                        <Switch>
                            <Route path="/project/:name" component={DataExplorer} />
                        </Switch>
                    </div>
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
