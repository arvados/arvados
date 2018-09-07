// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { connect, DispatchProp } from "react-redux";
import { Route, Switch } from "react-router";
import { User } from "~/models/user";
import { RootState } from "~/store/store";
import { MainAppBar } from '~/views-components/main-app-bar/main-app-bar';
import { push } from 'react-router-redux';
import { ProjectPanel } from "~/views/project-panel/project-panel";
import { DetailsPanel } from '~/views-components/details-panel/details-panel';
import { ArvadosTheme } from '~/common/custom-theme';
import { detailsPanelActions } from "~/store/details-panel/details-panel-action";
import { ContextMenu } from "~/views-components/context-menu/context-menu";
import { FavoritePanel } from "../favorite-panel/favorite-panel";
import { CurrentTokenDialog } from '~/views-components/current-token-dialog/current-token-dialog';
import { Snackbar } from '~/views-components/snackbar/snackbar';
import { CollectionPanel } from '../collection-panel/collection-panel';
import { AuthService } from "~/services/auth-service/auth-service";
import { RenameFileDialog } from '~/views-components/rename-file-dialog/rename-file-dialog';
import { FileRemoveDialog } from '~/views-components/file-remove-dialog/file-remove-dialog';
import { MultipleFilesRemoveDialog } from '~/views-components/file-remove-dialog/multiple-files-remove-dialog';
import { Routes } from '~/routes/routes';
import { SidePanel } from '~/views-components/side-panel/side-panel';
import { ProcessPanel } from '~/views/process-panel/process-panel';
import { ProcessLogPanel } from '~/views/process-log-panel/process-log-panel';
import { CreateProjectDialog } from '~/views-components/dialog-forms/create-project-dialog';
import { CreateCollectionDialog } from '~/views-components/dialog-forms/create-collection-dialog';
import { CopyCollectionDialog } from '~/views-components/dialog-forms/copy-collection-dialog';
import { UpdateCollectionDialog } from '~/views-components/dialog-forms/update-collection-dialog';
import { UpdateProjectDialog } from '~/views-components/dialog-forms/update-project-dialog';
import { MoveProjectDialog } from '~/views-components/dialog-forms/move-project-dialog';
import { MoveCollectionDialog } from '~/views-components/dialog-forms/move-collection-dialog';
import { FilesUploadCollectionDialog } from '~/views-components/dialog-forms/files-upload-collection-dialog';
import { PartialCopyCollectionDialog } from '~/views-components/dialog-forms/partial-copy-collection-dialog';

import { TrashPanel } from "~/views/trash-panel/trash-panel";
import { MainContentBar } from '../../views-components/main-content-bar/main-content-bar';
import { Grid } from '@material-ui/core';

type CssRules = 'root' | 'contentWrapper' | 'content' | 'appBar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflow: 'hidden',
        width: '100vw',
        height: '100vh'
    },
    contentWrapper: {
        background: theme.palette.background.default,
        minWidth: 0,
    },
    content: {
        minWidth: 0,
        overflow: 'auto',
        paddingLeft: theme.spacing.unit * 3,
        paddingRight: theme.spacing.unit * 3,
    },
    appBar: {
        zIndex: 1,
    }
});

interface WorkbenchDataProps {
    user?: User;
    currentToken?: string;
}

interface WorkbenchGeneralProps {
    authService: AuthService;
    buildInfo: string;
}

type WorkbenchProps = WorkbenchDataProps & WorkbenchGeneralProps & DispatchProp<any> & WithStyles<CssRules>;

interface WorkbenchState {
    searchText: string;
}

export const Workbench = withStyles(styles)(
    connect<WorkbenchDataProps>(
        (state: RootState) => ({
            user: state.auth.user,
            currentToken: state.auth.apiToken,
        })
    )(
        class extends React.Component<WorkbenchProps, WorkbenchState> {
            state = {
                searchText: "",
            };

            render() {
                return <>
                    <Grid
                        container
                        direction="column"
                        className={this.props.classes.root}>
                        <Grid className={this.props.classes.appBar}>
                            <MainAppBar
                                searchText={this.state.searchText}
                                user={this.props.user}
                                onSearch={this.onSearch}
                                buildInfo={this.props.buildInfo} />
                        </Grid>
                        {this.props.user &&
                            <Grid
                                container
                                item
                                xs
                                alignItems="stretch"
                                wrap="nowrap">
                                <Grid item>
                                    <SidePanel />
                                </Grid>
                                <Grid
                                    container
                                    item
                                    xs
                                    component="main"
                                    direction="column"
                                    className={this.props.classes.contentWrapper}>
                                    <Grid item>
                                        <MainContentBar />
                                    </Grid>
                                    <Grid item xs className={this.props.classes.content}>
                                        <Switch>
                                            <Route path={Routes.PROJECTS} component={ProjectPanel} />
                                            <Route path={Routes.COLLECTIONS} component={CollectionPanel} />
                                            <Route path={Routes.FAVORITES} component={FavoritePanel} />
                                            <Route path={Routes.PROCESSES} component={ProcessPanel} />
                                            <Route path={Routes.TRASH} component={TrashPanel} />
                                            <Route path={Routes.PROCESS_LOGS} component={ProcessLogPanel} />
                                        </Switch>
                                    </Grid>
                                </Grid>
                                <Grid item>
                                    <DetailsPanel />
                                </Grid>
                            </Grid>}
                    </Grid>
                    <ContextMenu />
                    <Snackbar />
                    <CreateProjectDialog />
                    <CreateCollectionDialog />
                    <RenameFileDialog />
                    <PartialCopyCollectionDialog />
                    <FileRemoveDialog />
                    <CopyCollectionDialog />
                    <FileRemoveDialog />
                    <MultipleFilesRemoveDialog />
                    <UpdateCollectionDialog />
                    <FilesUploadCollectionDialog />
                    <UpdateProjectDialog />
                    <MoveCollectionDialog />
                    <MoveProjectDialog />
                    <CurrentTokenDialog />
                </>;
            }

            onSearch = (searchText: string) => {
                this.setState({ searchText });
                this.props.dispatch(push(`/search?q=${searchText}`));
            }

            toggleDetailsPanel = () => {
                this.props.dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
            }

        }
    )
);
