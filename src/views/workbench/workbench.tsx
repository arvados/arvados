// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch, RouteComponentProps } from "react-router";
import { login, logout } from "../../store/auth/auth-action";
import { User } from "../../models/user";
import { RootState } from "../../store/store";
import { MainAppBar, MainAppBarActionProps, MainAppBarMenuItem } from '../../views-components/main-app-bar/main-app-bar';
import { Breadcrumb } from '../../components/breadcrumbs/breadcrumbs';
import { push } from 'react-router-redux';
import { ProjectTree } from '../../views-components/project-tree/project-tree';
import { TreeItem } from "../../components/tree/tree";
import { getTreePath } from '../../store/project/project-reducer';
import { sidePanelActions } from '../../store/side-panel/side-panel-action';
import { SidePanel, SidePanelItem } from '../../components/side-panel/side-panel';
import { ItemMode, setProjectItem } from "../../store/navigation/navigation-action";
import { projectActions } from "../../store/project/project-action";
import { collectionCreateActions } from '../../store/collections/creator/collection-creator-action';
import { ProjectPanel } from "../project-panel/project-panel";
import { DetailsPanel } from '../../views-components/details-panel/details-panel';
import { ArvadosTheme } from '../../common/custom-theme';
import { CreateProjectDialog } from "../../views-components/create-project-dialog/create-project-dialog";

import { detailsPanelActions, loadDetails } from "../../store/details-panel/details-panel-action";
import { contextMenuActions } from "../../store/context-menu/context-menu-actions";
import { SidePanelIdentifiers } from '../../store/side-panel/side-panel-reducer';
import { ProjectResource } from '../../models/project';
import { ResourceKind } from '../../models/resource';
import { ContextMenu, ContextMenuKind } from "../../views-components/context-menu/context-menu";
import { FavoritePanel } from "../favorite-panel/favorite-panel";
import { CurrentTokenDialog } from '../../views-components/current-token-dialog/current-token-dialog';
import { Snackbar } from '../../views-components/snackbar/snackbar';
import { favoritePanelActions } from '../../store/favorite-panel/favorite-panel-action';
import { CreateCollectionDialog } from '../../views-components/create-collection-dialog/create-collection-dialog';
import { CollectionPanel } from '../collection-panel/collection-panel';
import { loadCollection } from '../../store/collection-panel/collection-panel-action';
import { getCollectionUrl } from '../../models/collection';
import { UpdateCollectionDialog } from '../../views-components/update-collection-dialog/update-collection-dialog.';
import { AuthService } from "../../services/auth-service/auth-service";
import { RenameFileDialog } from '../../views-components/rename-file-dialog/rename-file-dialog';
import { FileRemoveDialog } from '../../views-components/file-remove-dialog/file-remove-dialog';
import { MultipleFilesRemoveDialog } from '../../views-components/file-remove-dialog/multiple-files-remove-dialog';

const DRAWER_WITDH = 240;
const APP_BAR_HEIGHT = 100;

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
        width: DRAWER_WITDH,
        display: 'flex',
        flexDirection: 'column',
    },
    contentWrapper: {
        backgroundColor: theme.palette.background.default,
        display: "flex",
        flexGrow: 1,
        minWidth: 0,
        paddingTop: APP_BAR_HEIGHT
    },
    content: {
        padding: `${theme.spacing.unit}px ${theme.spacing.unit * 3}px`,
        overflowY: "auto",
        flexGrow: 1
    },
    toolbar: theme.mixins.toolbar
});

interface WorkbenchDataProps {
    projects: Array<TreeItem<ProjectResource>>;
    currentProjectId: string;
    user?: User;
    currentToken?: string;
    sidePanelItems: SidePanelItem[];
}

interface WorkbenchServiceProps {
    authService: AuthService;
}

interface WorkbenchActionProps {
}

type WorkbenchProps = WorkbenchDataProps & WorkbenchServiceProps & WorkbenchActionProps & DispatchProp<any> & WithStyles<CssRules>;

interface NavBreadcrumb extends Breadcrumb {
    itemId: string;
}

interface NavMenuItem extends MainAppBarMenuItem {
    action: () => void;
}

interface WorkbenchState {
    isCurrentTokenDialogOpen: boolean;
    anchorEl: any;
    searchText: string;
    menuItems: {
        accountMenu: NavMenuItem[],
        helpMenu: NavMenuItem[],
        anonymousMenu: NavMenuItem[]
    };
}

export const Workbench = withStyles(styles)(
    connect<WorkbenchDataProps>(
        (state: RootState) => ({
            projects: state.projects.items,
            currentProjectId: state.projects.currentItemId,
            user: state.auth.user,
            currentToken: state.auth.apiToken,
            sidePanelItems: state.sidePanel
        })
    )(
        class extends React.Component<WorkbenchProps, WorkbenchState> {
            state = {
                isCreationDialogOpen: false,
                isCurrentTokenDialogOpen: false,
                anchorEl: null,
                searchText: "",
                breadcrumbs: [],
                menuItems: {
                    accountMenu: [
                        {
                            label: 'Current token',
                            action: () => this.toggleCurrentTokenModal()
                        },
                        {
                            label: "Logout",
                            action: () => this.props.dispatch(logout())
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
                            action: () => this.props.dispatch(login())
                        }
                    ]
                }
            };

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
                                    onContextMenu={(event) => this.openContextMenu(event, {
                                        uuid: this.props.authService.getUuid() || "",
                                        name: "",
                                        kind: ContextMenuKind.ROOT_PROJECT
                                    })}>
                                    <ProjectTree
                                        projects={this.props.projects}
                                        toggleOpen={itemId => this.props.dispatch(setProjectItem(itemId, ItemMode.OPEN))}
                                        onContextMenu={(event, item) => this.openContextMenu(event, {
                                            uuid: item.data.uuid,
                                            name: item.data.name,
                                            kind: ContextMenuKind.PROJECT
                                        })}
                                        toggleActive={itemId => {
                                            this.props.dispatch(setProjectItem(itemId, ItemMode.ACTIVE));
                                            this.props.dispatch(loadDetails(itemId, ResourceKind.PROJECT));
                                            this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(SidePanelIdentifiers.PROJECTS));
                                        }} />
                                </SidePanel>
                            </Drawer>}
                        <main className={classes.contentWrapper}>
                            <div className={classes.content}>
                                <Switch>
                                    <Route path="/projects/:id" render={this.renderProjectPanel} />
                                    <Route path="/favorites" render={this.renderFavoritePanel} />
                                    <Route path="/collections/:id" render={this.renderCollectionPanel} />
                                </Switch>
                            </div>
                            {user && <DetailsPanel />}
                        </main>
                        <ContextMenu />
                        <Snackbar />
                        <CreateProjectDialog />
                        <CreateCollectionDialog />
                        <RenameFileDialog />
                        <FileRemoveDialog />
                        <MultipleFilesRemoveDialog />
                        <UpdateCollectionDialog />
                        <CurrentTokenDialog
                            currentToken={this.props.currentToken}
                            open={this.state.isCurrentTokenDialogOpen}
                            handleClose={this.toggleCurrentTokenModal} />
                    </div>
                );
            }

            renderCollectionPanel = (props: RouteComponentProps<{ id: string }>) => <CollectionPanel
                onItemRouteChange={(collectionId) => this.props.dispatch<any>(loadCollection(collectionId, ResourceKind.COLLECTION))}
                onContextMenu={(event, item) => {
                    this.openContextMenu(event, {
                        uuid: item.uuid,
                        name: item.name,
                        kind: ContextMenuKind.COLLECTION
                    });
                }}
                {...props} />

            renderProjectPanel = (props: RouteComponentProps<{ id: string }>) => <ProjectPanel
                onItemRouteChange={itemId => this.props.dispatch(setProjectItem(itemId, ItemMode.ACTIVE))}
                onContextMenu={(event, item) => {

                    const kind = item.kind === ResourceKind.PROJECT ? ContextMenuKind.PROJECT : ContextMenuKind.RESOURCE;
                    this.openContextMenu(event, {
                        uuid: item.uuid,
                        name: item.name,
                        kind
                    });
                }}
                onProjectCreationDialogOpen={this.handleProjectCreationDialogOpen}
                onCollectionCreationDialogOpen={this.handleCollectionCreationDialogOpen}
                onItemClick={item => {
                    this.props.dispatch(loadDetails(item.uuid, item.kind as ResourceKind));
                }}
                onItemDoubleClick={item => {
                    switch (item.kind) {
                        case ResourceKind.COLLECTION:
                            this.props.dispatch(loadCollection(item.uuid, item.kind as ResourceKind));
                            this.props.dispatch(push(getCollectionUrl(item.uuid)));
                        default: 
                            this.props.dispatch(setProjectItem(item.uuid, ItemMode.ACTIVE));
                            this.props.dispatch(loadDetails(item.uuid, item.kind as ResourceKind));
                    }

                }}
                {...props} />

            renderFavoritePanel = (props: RouteComponentProps<{ id: string }>) => <FavoritePanel
                onItemRouteChange={() => this.props.dispatch(favoritePanelActions.REQUEST_ITEMS())}
                onContextMenu={(event, item) => {
                    const kind = item.kind === ResourceKind.PROJECT ? ContextMenuKind.PROJECT : ContextMenuKind.RESOURCE;
                    this.openContextMenu(event, {
                        uuid: item.uuid,
                        name: item.name,
                        kind,
                    });
                }}
                onDialogOpen={this.handleProjectCreationDialogOpen}
                onItemClick={item => {
                    this.props.dispatch(loadDetails(item.uuid, item.kind as ResourceKind));
                }}
                onItemDoubleClick={item => {
                    switch (item.kind) {
                        case ResourceKind.COLLECTION:
                            this.props.dispatch(loadCollection(item.uuid, item.kind as ResourceKind));
                            this.props.dispatch(push(getCollectionUrl(item.uuid)));
                        default:
                            this.props.dispatch(loadDetails(item.uuid, ResourceKind.PROJECT));
                            this.props.dispatch(setProjectItem(item.uuid, ItemMode.ACTIVE));
                            this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(SidePanelIdentifiers.PROJECTS));
                    }

                }}
                {...props} />

            mainAppBarActions: MainAppBarActionProps = {
                onBreadcrumbClick: ({ itemId }: NavBreadcrumb) => {
                    this.props.dispatch(setProjectItem(itemId, ItemMode.BOTH));
                    this.props.dispatch(loadDetails(itemId, ResourceKind.PROJECT));
                },
                onSearch: searchText => {
                    this.setState({ searchText });
                    this.props.dispatch(push(`/search?q=${searchText}`));
                },
                onMenuItemClick: (menuItem: NavMenuItem) => menuItem.action(),
                onDetailsPanelToggle: () => {
                    this.props.dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
                },
                onContextMenu: (event: React.MouseEvent<HTMLElement>, breadcrumb: NavBreadcrumb) => {
                    this.openContextMenu(event, {
                        uuid: breadcrumb.itemId,
                        name: breadcrumb.label,
                        kind: ContextMenuKind.PROJECT
                    });
                }
            };

            toggleSidePanelOpen = (itemId: string) => {
                this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(itemId));
            }

            toggleSidePanelActive = (itemId: string) => {
                this.props.dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_ACTIVE(itemId));
                this.props.dispatch(projectActions.RESET_PROJECT_TREE_ACTIVITY(itemId));
                const panelItem = this.props.sidePanelItems.find(it => it.id === itemId);
                if (panelItem && panelItem.activeAction) {
                    panelItem.activeAction(this.props.dispatch);
                }
            }

            handleProjectCreationDialogOpen = (itemUuid: string) => {
                this.props.dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: itemUuid }));
            }

            handleCollectionCreationDialogOpen = (itemUuid: string) => {
                this.props.dispatch(collectionCreateActions.OPEN_COLLECTION_CREATOR({ ownerUuid: itemUuid }));
            }

            openContextMenu = (event: React.MouseEvent<HTMLElement>, resource: { name: string; uuid: string; kind: ContextMenuKind; }) => {
                event.preventDefault();
                this.props.dispatch(
                    contextMenuActions.OPEN_CONTEXT_MENU({
                        position: { x: event.clientX, y: event.clientY },
                        resource
                    })
                );
            }

            toggleCurrentTokenModal = () => {
                this.setState({ isCurrentTokenDialogOpen: !this.state.isCurrentTokenDialogOpen });
            }
        }
    )
);
