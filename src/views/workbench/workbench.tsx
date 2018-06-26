// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, Theme, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch } from "react-router";
import authActions from "../../store/auth/auth-action";
import dataExplorerActions from "../../store/data-explorer/data-explorer-action";
import { User } from "../../models/user";
import { RootState } from "../../store/store";
import MainAppBar, { MainAppBarActionProps, MainAppBarMenuItem } from '../../views-components/main-app-bar/main-app-bar';
import { Breadcrumb } from '../../components/breadcrumbs/breadcrumbs';
import { push } from 'react-router-redux';
import projectActions, { getProjectList } from "../../store/project/project-action";
import ProjectTree from '../../views-components/project-tree/project-tree';
import { TreeItem, TreeItemStatus } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { getTreePath, findTreeItem } from '../../store/project/project-reducer';
import ProjectExplorer, { PROJECT_EXPLORER_ID } from '../../views-components/project-explorer/project-explorer';
import { ProjectExplorerItem, mapProjectTreeItem } from '../../views-components/project-explorer/project-explorer-item';
import sidePanelActions from '../../store/side-panel/side-panel-action';
import SidePanel, { SidePanelItem } from '../../components/side-panel/side-panel';

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
        display: 'flex',
        flexDirection: 'column',
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
    sidePanelItems: SidePanelItem[];
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
            this.toggleProjectTreeItemOpen(itemId, status);
        },
        onSearch: searchText => {
            this.setState({ searchText });
            this.props.dispatch(push(`/search?q=${searchText}`));
        },
        onMenuItemClick: (menuItem: NavMenuItem) => menuItem.action()
    };

    toggleProjectTreeItemOpen = (itemId: string, status: TreeItemStatus) => {
        if (status === TreeItemStatus.Loaded) {
            this.openProjectItem(itemId);
            this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(itemId));
            this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
        } else {
            this.props.dispatch<any>(getProjectList(itemId))
                .then(() => {
                    this.openProjectItem(itemId);
                    this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN(itemId));
                    this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
                });
        }
    }

    toggleProjectTreeItemActive = (itemId: string, status: TreeItemStatus) => {
        if (status === TreeItemStatus.Loaded) {
            this.openProjectItem(itemId);
            this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
            this.props.dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(itemId));
        } else {
            this.props.dispatch<any>(getProjectList(itemId))
                .then(() => {
                    this.openProjectItem(itemId);
                    this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
                    this.props.dispatch(sidePanelActions.RESET_SIDE_PANEL_ACTIVITY(itemId));
                });
        }
    }

    toggleSidePanelOpen = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(itemId));
    }

    toggleSidePanelActive = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(itemId));
        this.props.dispatch(projectActions.RESET_PROJECT_TREE_ACTIVITY(itemId));
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
        this.props.dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_ACTIVE(itemId));
        this.props.dispatch(push(`/project/${itemId}`));

        const project = findTreeItem(this.props.projects, itemId);
        const items: ProjectExplorerItem[] = project && project.items
            ? project.items.map(mapProjectTreeItem)
            : [];
        this.props.dispatch(dataExplorerActions.SET_ITEMS({ id: PROJECT_EXPLORER_ID, items }));
    }

    render() {
        const { classes, user, projects, sidePanelItems } = this.props;
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
                        <SidePanel
                            toggleOpen={this.toggleSidePanelOpen}
                            toggleActive={this.toggleSidePanelActive}
                            sidePanelItems={sidePanelItems}>
                            <ProjectTree
                                projects={projects}
                                toggleOpen={this.toggleProjectTreeItemOpen}
                                toggleActive={this.toggleProjectTreeItemActive} />
                        </SidePanel>
                    </Drawer>}
                <main className={classes.contentWrapper}>
                    <div className={classes.content}>
                        <Switch>
                            <Route path="/project/:name" component={ProjectExplorer} />
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
        user: state.auth.user,
        sidePanelItems: state.sidePanel,
    })
)(
    withStyles(styles)(Workbench)
);
