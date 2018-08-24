// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch, Redirect } from "react-router";
import { login, logout } from "~/store/auth/auth-action";
import { User } from "~/models/user";
import { RootState } from "~/store/store";
import { MainAppBar, MainAppBarActionProps, MainAppBarMenuItem } from '~/views-components/main-app-bar/main-app-bar';
import { Breadcrumb } from '~/components/breadcrumbs/breadcrumbs';
import { push } from 'react-router-redux';
import { TreeItem } from "~/components/tree/tree";
import { getTreePath } from '~/store/project/project-reducer';
import { ProjectPanel } from "~/views/project-panel/project-panel";
import { DetailsPanel } from '~/views-components/details-panel/details-panel';
import { ArvadosTheme } from '~/common/custom-theme';
import { CreateProjectDialog } from "~/views-components/create-project-dialog/create-project-dialog";
import { detailsPanelActions, loadDetailsPanel } from "~/store/details-panel/details-panel-action";
import { openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ProjectResource, getProjectUrl } from '~/models/project';
import { ContextMenu, ContextMenuKind } from "~/views-components/context-menu/context-menu";
import { FavoritePanel } from "../favorite-panel/favorite-panel";
import { CurrentTokenDialog } from '~/views-components/current-token-dialog/current-token-dialog';
import { Snackbar } from '~/views-components/snackbar/snackbar';
import { CreateCollectionDialog } from '~/views-components/create-collection-dialog/create-collection-dialog';
import { CollectionPanel } from '../collection-panel/collection-panel';
import { UpdateCollectionDialog } from '~/views-components/update-collection-dialog/update-collection-dialog.';
import { UpdateProjectDialog } from '~/views-components/update-project-dialog/update-project-dialog';
import { AuthService } from "~/services/auth-service/auth-service";
import { RenameFileDialog } from '~/views-components/rename-file-dialog/rename-file-dialog';
import { FileRemoveDialog } from '~/views-components/file-remove-dialog/file-remove-dialog';
import { MultipleFilesRemoveDialog } from '~/views-components/file-remove-dialog/multiple-files-remove-dialog';
import { DialogCollectionCreateWithSelectedFile } from '~/views-components/create-collection-dialog-with-selected/create-collection-dialog-with-selected';
import { UploadCollectionFilesDialog } from '~/views-components/upload-collection-files-dialog/upload-collection-files-dialog';
import { ProjectCopyDialog } from '~/views-components/project-copy-dialog/project-copy-dialog';
import { CollectionPartialCopyDialog } from '~/views-components/collection-partial-copy-dialog/collection-partial-copy-dialog';
import { MoveProjectDialog } from '~/views-components/move-project-dialog/move-project-dialog';
import { MoveCollectionDialog } from '~/views-components/move-collection-dialog/move-collection-dialog';
import { SidePanel } from '~/views-components/side-panel/side-panel';
import { Routes } from '~/routes/routes';
import { navigateToResource } from '~/store/navigation/navigation-action';

const APP_BAR_HEIGHT = 100;

type CssRules = 'root' | 'appBar' | 'content' | 'contentWrapper';

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
        flexGrow: 1,
        position: 'relative'
    },
});

interface WorkbenchDataProps {
    projects: Array<TreeItem<ProjectResource>>;
    currentProjectId: string;
    user?: User;
    currentToken?: string;
}

interface WorkbenchGeneralProps {
    authService: AuthService;
    buildInfo: string;
}

interface WorkbenchActionProps {
}

type WorkbenchProps = WorkbenchDataProps & WorkbenchGeneralProps & WorkbenchActionProps & DispatchProp<any> & WithStyles<CssRules>;

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
        })
    )(
        class extends React.Component<WorkbenchProps, WorkbenchState> {
            state = {
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

                const rootProjectUuid = this.props.authService.getUuid();

                const { classes, user } = this.props;
                return (
                    <div className={classes.root}>
                        <div className={classes.appBar}>
                            <MainAppBar
                                breadcrumbs={breadcrumbs}
                                searchText={this.state.searchText}
                                user={this.props.user}
                                menuItems={this.state.menuItems}
                                buildInfo={this.props.buildInfo}
                                {...this.mainAppBarActions} />
                        </div>
                        {user && <SidePanel />}
                        <main className={classes.contentWrapper}>
                            <div className={classes.content}>
                                <Switch>
                                    <Route path={Routes.PROJECTS} component={ProjectPanel} />
                                    <Route path={Routes.COLLECTIONS} component={CollectionPanel} />
                                    <Route path={Routes.FAVORITES} component={FavoritePanel} />
                                </Switch>
                            </div>
                            {user && <DetailsPanel />}
                        </main>
                        <ContextMenu />
                        <Snackbar />
                        <CreateProjectDialog />
                        <CreateCollectionDialog />
                        <RenameFileDialog />
                        <CollectionPartialCopyDialog />
                        <DialogCollectionCreateWithSelectedFile />
                        <FileRemoveDialog />
                        <ProjectCopyDialog />
                        <MultipleFilesRemoveDialog />
                        <UpdateCollectionDialog />
                        <UploadCollectionFilesDialog />
                        <UpdateProjectDialog />
                        <MoveCollectionDialog />
                        <MoveProjectDialog />
                        <CurrentTokenDialog
                            currentToken={this.props.currentToken}
                            open={this.state.isCurrentTokenDialogOpen}
                            handleClose={this.toggleCurrentTokenModal} />
                    </div>
                );
            }

            mainAppBarActions: MainAppBarActionProps = {
                onBreadcrumbClick: ({ itemId }: NavBreadcrumb) => {
                    this.props.dispatch(navigateToResource(itemId));
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
                    this.props.dispatch<any>(openContextMenu(event, {
                        uuid: breadcrumb.itemId,
                        name: breadcrumb.label,
                        kind: ContextMenuKind.PROJECT
                    }));
                }
            };

            toggleCurrentTokenModal = () => {
                this.setState({ isCurrentTokenDialogOpen: !this.state.isCurrentTokenDialogOpen });
            }
        }
    )
);
