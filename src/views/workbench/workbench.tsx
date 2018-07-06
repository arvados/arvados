// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch, RouteComponentProps, withRouter } from "react-router";
import authActions from "../../store/auth/auth-action";
import dataExplorerActions from "../../store/data-explorer/data-explorer-action";
import { User } from "../../models/user";
import { RootState } from "../../store/store";
import MainAppBar, {
    MainAppBarActionProps,
    MainAppBarMenuItem
} from '../../views-components/main-app-bar/main-app-bar';
import { Breadcrumb } from '../../components/breadcrumbs/breadcrumbs';
import { push } from 'react-router-redux';
import ProjectTree from '../../views-components/project-tree/project-tree';
import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { getTreePath, findTreeItem } from '../../store/project/project-reducer';
import sidePanelActions from '../../store/side-panel/side-panel-action';
import SidePanel, { SidePanelItem } from '../../components/side-panel/side-panel';
import { ResourceKind } from "../../models/resource";
import { ItemMode, setProjectItem } from "../../store/navigation/navigation-action";
import projectActions from "../../store/project/project-action";
import ProjectPanel from "../project-panel/project-panel";
import { sidePanelData } from '../../store/side-panel/side-panel-reducer';
import DetailsPanel from '../../views-components/details-panel/details-panel';
import { ArvadosTheme } from '../../common/custom-theme';

const drawerWidth = 240;
const appBarHeight = 100;

type CssRules = 'root' | 'appBar' | 'drawerPaper' | 'content' | 'contentWrapper' | 'toolbar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
        padding: `${theme.spacing.unit}px ${theme.spacing.unit * 3}px`,
        overflowY: "auto",
        flexGrow: 1
    },
    toolbar: theme.mixins.toolbar
});

interface WorkbenchDataProps {
    projects: Array<TreeItem<Project>>;
    currentProjectId: string;
    user?: User;
    sidePanelItems: SidePanelItem[];
}

interface WorkbenchActionProps {
}

type WorkbenchProps = WorkbenchDataProps & WorkbenchActionProps & DispatchProp & WithStyles<CssRules>;

interface NavBreadcrumb extends Breadcrumb {
    itemId: string;
}

interface NavMenuItem extends MainAppBarMenuItem {
    action: () => void;
}

interface WorkbenchState {
    anchorEl: any;
    searchText: string;
    menuItems: {
        accountMenu: NavMenuItem[],
        helpMenu: NavMenuItem[],
        anonymousMenu: NavMenuItem[]
    };
    isDetailsPanelOpened: boolean;
    detailsPanelHeader: React.ComponentType<{}> | undefined;
    detailsPanelDetails: React.ComponentType<{}> | undefined;
    detailsPanelActivity: React.ComponentType<{}> | undefined;
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
        },
        isDetailsPanelOpened: false,
        detailsPanelHeader: undefined,
        detailsPanelDetails: undefined,
        detailsPanelActivity: undefined,
    };

    mainAppBarActions: MainAppBarActionProps = {
        onBreadcrumbClick: ({ itemId }: NavBreadcrumb) => {
            this.props.dispatch<any>(setProjectItem(itemId, ItemMode.BOTH));
        },
        onSearch: searchText => {
            this.setState({ searchText });
            this.props.dispatch(push(`/search?q=${searchText}`));
        },
        onMenuItemClick: (menuItem: NavMenuItem) => menuItem.action(),
        onDetailsPanelToggle: () => {
            this.setState(prev => ({ isDetailsPanelOpened: !prev.isDetailsPanelOpened }));
        }
    };

    toggleSidePanelOpen = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(itemId));
    }

    toggleSidePanelActive = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(itemId));
        this.props.dispatch(projectActions.RESET_PROJECT_TREE_ACTIVITY(itemId));
    }

    render() {
        const path = getTreePath(this.props.projects, this.props.currentProjectId);
        const breadcrumbs = path.map(item => ({
            label: item.data.name,
            itemId: item.data.uuid,
            status: item.status
        }));

        const { classes, user } = this.props;
        return (
            <div className={classes.root}>
                <div className={classes.appBar}>
                    <MainAppBar
                        breadcrumbs={breadcrumbs}
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
                            sidePanelItems={this.props.sidePanelItems}>
                            <ProjectTree
                                projects={this.props.projects}
                                toggleOpen={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.OPEN))}
                                toggleActive={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.ACTIVE))}
                            />
                        </SidePanel>
                    </Drawer>}
                <main className={classes.contentWrapper}>
                    <div className={classes.content}>
                        <Switch>
                            <Route path="/projects/:id" render={this.renderProjectPanel} />
                        </Switch>
                    </div>
                    <DetailsPanel
                        renderHeader={this.state.detailsPanelHeader}
                        renderDetails={this.state.detailsPanelDetails}
                        renderActivity={this.state.detailsPanelActivity}
                        isOpened={this.state.isDetailsPanelOpened} 
                        onCloseDrawer={this.mainAppBarActions.onDetailsPanelToggle} />
                </main>
            </div>
        );
    }

    renderProjectPanel = (props: RouteComponentProps<{ id: string }>) => <ProjectPanel
        onItemRouteChange={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.ACTIVE))}
        onItemClick={item => this.props.dispatch<any>(setProjectItem(item.uuid, ItemMode.ACTIVE))}
        {...props} />

}

export default connect<WorkbenchDataProps>(
    (state: RootState) => ({
        projects: state.projects.items,
        currentProjectId: state.projects.currentItemId,
        user: state.auth.user,
        sidePanelItems: state.sidePanel
    })
)(
    withStyles(styles)(Workbench)
);
