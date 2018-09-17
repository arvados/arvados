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
import { CopyProcessDialog } from '~/views-components/dialog-forms/copy-process-dialog';
import { UpdateCollectionDialog } from '~/views-components/dialog-forms/update-collection-dialog';
import { UpdateProcessDialog } from '~/views-components/dialog-forms/update-process-dialog';
import { UpdateProjectDialog } from '~/views-components/dialog-forms/update-project-dialog';
import { MoveProcessDialog } from '~/views-components/dialog-forms/move-process-dialog';
import { MoveProjectDialog } from '~/views-components/dialog-forms/move-project-dialog';
import { MoveCollectionDialog } from '~/views-components/dialog-forms/move-collection-dialog';
import { FilesUploadCollectionDialog } from '~/views-components/dialog-forms/files-upload-collection-dialog';
import { PartialCopyCollectionDialog } from '~/views-components/dialog-forms/partial-copy-collection-dialog';
import { TrashPanel } from "~/views/trash-panel/trash-panel";
import { MainContentBar } from '~/views-components/main-content-bar/main-content-bar';
import { Grid } from '@material-ui/core';
import { SharedWithMePanel } from '../shared-with-me-panel/shared-with-me-panel';
import { ProcessCommandDialog } from '~/views-components/process-command-dialog/process-command-dialog';
import SplitterLayout from 'react-splitter-layout';

type CssRules = 'root' | 'container' | 'splitter' | 'asidePanel' | 'contentWrapper' | 'content' | 'appBar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflow: 'hidden',
        width: '100vw',
        height: '100vh'
    },
    container: {
        position: 'relative'
    },
    splitter: {
        '& > .layout-splitter': {
            width: '2px'
        }
    },
    asidePanel: {
        height: '100%',
        background: theme.palette.background.default
    },
    contentWrapper: {
        background: theme.palette.background.default,
        minWidth: 0,
    },
    content: {
        minWidth: 0,
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
                const { classes } = this.props;
                return <>
                    <Grid container direction="column" className={classes.root}>
                        <Grid className={classes.appBar}>
                            <MainAppBar
                                searchText={this.state.searchText}
                                user={this.props.user}
                                onSearch={this.onSearch}
                                buildInfo={this.props.buildInfo} />
                        </Grid>
                        {this.props.user &&
                            <Grid container item xs alignItems="stretch" wrap="nowrap">
                                <Grid container item className={classes.container}>
                                <SplitterLayout customClassName={classes.splitter} percentage={true}
                                    primaryIndex={0} primaryMinSize={20} secondaryInitialSize={80} secondaryMinSize={40}>
                                        <Grid container item xs component='aside' direction='column' className={classes.asidePanel}>
                                            <SidePanel />
                                        </Grid>
                                        <Grid container item xs component="main" direction="column" className={classes.contentWrapper}>
                                            <Grid item>
                                                <MainContentBar />
                                            </Grid>
                                            <Grid item xs className={classes.content}>
                                                <Switch>
                                                    <Route path={Routes.PROJECTS} component={ProjectPanel} />
                                                    <Route path={Routes.COLLECTIONS} component={CollectionPanel} />
                                                    <Route path={Routes.FAVORITES} component={FavoritePanel} />
                                                    <Route path={Routes.PROCESSES} component={ProcessPanel} />
                                                    <Route path={Routes.TRASH} component={TrashPanel} />
                                                    <Route path={Routes.PROCESS_LOGS} component={ProcessLogPanel} />
                                                    <Route path={Routes.SHARED_WITH_ME} component={SharedWithMePanel} />
                                                </Switch>
                                            </Grid>
                                        </Grid>
                                    </SplitterLayout>
                                </Grid>
                                <Grid item>
                                    <DetailsPanel />
                                </Grid>
                            </Grid>
                        }
                    </Grid>
                    <ContextMenu />
                    <CopyCollectionDialog />
                    <CopyProcessDialog />
                    <CreateCollectionDialog />
                    <CreateProjectDialog />
                    <CurrentTokenDialog />
                    <FileRemoveDialog />
                    <FileRemoveDialog />
                    <FilesUploadCollectionDialog />
                    <MoveCollectionDialog />
                    <MoveProcessDialog />
                    <MoveProjectDialog />
                    <MultipleFilesRemoveDialog />
                    <PartialCopyCollectionDialog />
                    <ProcessCommandDialog />
                    <RenameFileDialog />
                    <Snackbar />
                    <UpdateCollectionDialog />
                    <UpdateProcessDialog />
                    <UpdateProjectDialog />
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
