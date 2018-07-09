// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch, RouteComponentProps } from "react-router";
import authActions from "../../store/auth/auth-action";
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
import { getTreePath } from '../../store/project/project-reducer';
import sidePanelActions from '../../store/side-panel/side-panel-action';
import SidePanel, { SidePanelItem } from '../../components/side-panel/side-panel';
import { ItemMode, setProjectItem } from "../../store/navigation/navigation-action";
import projectActions from "../../store/project/project-action";
import ProjectPanel from "../project-panel/project-panel";
import DetailsPanel from '../../views-components/details-panel/details-panel';
import { ArvadosTheme } from '../../common/custom-theme';
import ContextMenu, { ContextMenuAction } from '../../components/context-menu/context-menu';
import { mockAnchorFromMouseEvent } from '../../components/popover/helpers';
import DialogProjectCreate from '../../components/dialog-create/dialog-project-create';

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
    contextMenu: {
        anchorEl?: HTMLElement;
    };
    isCreationDialogOpen: boolean;
    anchorEl: any;
    searchText: string;
    menuItems: {
        accountMenu: NavMenuItem[],
        helpMenu: NavMenuItem[],
        anonymousMenu: NavMenuItem[]
    };
    isDetailsPanelOpened: boolean;
}

class Workbench extends React.Component<WorkbenchProps, WorkbenchState> {
    state = {
        contextMenu: {
            anchorEl: undefined
        },
        isCreationDialogOpen: false,
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
        isDetailsPanelOpened: false
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
        },
        onContextMenu: (event: React.MouseEvent<HTMLElement>, breadcrumb: Breadcrumb) => {
            this.openContextMenu(event, breadcrumb);
        }
    };

    toggleSidePanelOpen = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(itemId));
    }

    toggleSidePanelActive = (itemId: string) => {
        this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(itemId));
        this.props.dispatch(projectActions.RESET_PROJECT_TREE_ACTIVITY(itemId));
    }

    handleCreationDialogOpen = () => {
        this.closeContextMenu();
        this.setState({ isCreationDialogOpen: true });
    }

    handleCreationDialogClose = () => {
        this.setState({ isCreationDialogOpen: false });
    }

    openContextMenu = (event: React.MouseEvent<HTMLElement>, item: any) => {
        event.preventDefault();
        this.setState({ contextMenu: { anchorEl: mockAnchorFromMouseEvent(event) } });
        console.log(item);
    }

    closeContextMenu = () => {
        this.setState({ contextMenu: {} });
    }

    openCreateDialog = (item: ContextMenuAction) =>
        item.openCreateDialog ? this.handleCreationDialogOpen() : void 0

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
                        {...this.mainAppBarActions} />
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
                            sidePanelItems={this.props.sidePanelItems}
                            onContextMenu={this.openContextMenu}>
                            <ProjectTree
                                projects={this.props.projects}
                                toggleOpen={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.OPEN))}
                                toggleActive={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.ACTIVE))}
                                onContextMenu={this.openContextMenu} />
                        </SidePanel>
                    </Drawer>}
                <main className={classes.contentWrapper}>
                    <div className={classes.content}>
                        <Switch>
                            <Route path="/projects/:id" render={this.renderProjectPanel} />
                        </Switch>
                    </div>
                    <DetailsPanel
                        isOpened={this.state.isDetailsPanelOpened}
                        onCloseDrawer={this.mainAppBarActions.onDetailsPanelToggle} />
                </main>
                <ContextMenu
                    anchorEl={this.state.contextMenu.anchorEl}
                    actions={contextMenuActions}
                    onActionClick={this.openCreateDialog}
                    onClose={this.closeContextMenu} />
                <DialogProjectCreate open={this.state.isCreationDialogOpen} handleClose={this.handleCreationDialogClose} />
            </div>
        );
    }

    renderProjectPanel = (props: RouteComponentProps<{ id: string }>) => <ProjectPanel
        onItemRouteChange={itemId => this.props.dispatch<any>(setProjectItem(itemId, ItemMode.ACTIVE))}
        onItemClick={item => this.props.dispatch<any>(setProjectItem(item.uuid, ItemMode.ACTIVE))}
        onContextMenu={this.openContextMenu}
        onDialogOpen={this.handleCreationDialogOpen}
        {...props} />
}

const contextMenuActions = [[{
    icon: "fas fa-plus fa-fw",
    name: "New project",
    openCreateDialog: true
}, {
    icon: "fas fa-users fa-fw",
    name: "Share"
}, {
    icon: "fas fa-sign-out-alt fa-fw",
    name: "Move to"
}, {
    icon: "fas fa-star fa-fw",
    name: "Add to favourite"
}, {
    icon: "fas fa-edit fa-fw",
    name: "Rename"
}, {
    icon: "fas fa-copy fa-fw",
    name: "Make a copy"
}, {
    icon: "fas fa-download fa-fw",
    name: "Download"
}], [{
    icon: "fas fa-trash-alt fa-fw",
    name: "Remove"
}
]];

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
