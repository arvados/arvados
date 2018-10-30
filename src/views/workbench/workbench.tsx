// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Route, Switch } from "react-router";
import { ProjectPanel } from "~/views/project-panel/project-panel";
import { DetailsPanel } from '~/views-components/details-panel/details-panel';
import { ArvadosTheme } from '~/common/custom-theme';
import { ContextMenu } from "~/views-components/context-menu/context-menu";
import { FavoritePanel } from "../favorite-panel/favorite-panel";
import { CurrentTokenDialog } from '~/views-components/current-token-dialog/current-token-dialog';
import { RichTextEditorDialog } from '~/views-components/rich-text-editor-dialog/rich-text-editor-dialog';
import { Snackbar } from '~/views-components/snackbar/snackbar';
import { CollectionPanel } from '../collection-panel/collection-panel';
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
import { ProcessCommandDialog } from '~/views-components/process-command-dialog/process-command-dialog';
import { MainContentBar } from '~/views-components/main-content-bar/main-content-bar';
import { Grid } from '@material-ui/core';
import { TrashPanel } from "~/views/trash-panel/trash-panel";
import { SharedWithMePanel } from '~/views/shared-with-me-panel/shared-with-me-panel';
import { RunProcessPanel } from '~/views/run-process-panel/run-process-panel';
import SplitterLayout from 'react-splitter-layout';
import { WorkflowPanel } from '~/views/workflow-panel/workflow-panel';
import { SearchResultsPanel } from '~/views/search-results-panel/search-results-panel';
import { SharingDialog } from '~/views-components/sharing-dialog/sharing-dialog';
import { AdvancedTabDialog } from '~/views-components/advanced-tab-dialog/advanced-tab-dialog';

type CssRules = 'root' | 'container' | 'splitter' | 'asidePanel' | 'contentWrapper' | 'content';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        paddingTop: theme.spacing.unit * 7,
        background: theme.palette.background.default
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
        paddingTop: theme.spacing.unit,
        height: '100%'
    },
    contentWrapper: {
        paddingTop: theme.spacing.unit,
        minWidth: 0
    },
    content: {
        minWidth: 0,
        paddingLeft: theme.spacing.unit * 3,
        paddingRight: theme.spacing.unit * 3,
    }
});

type WorkbenchPanelProps = WithStyles<CssRules>;

export const WorkbenchPanel =
    withStyles(styles)(({ classes }: WorkbenchPanelProps) =>
        <Grid container item xs className={classes.root}>
            <Grid container item xs className={classes.container}>
                <SplitterLayout customClassName={classes.splitter} percentage={true}
                    primaryIndex={0} primaryMinSize={10} secondaryInitialSize={90} secondaryMinSize={40}>
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
                                <Route path={Routes.RUN_PROCESS} component={RunProcessPanel} />
                                <Route path={Routes.WORKFLOWS} component={WorkflowPanel} />
                                <Route path={Routes.SEARCH_RESULTS} component={SearchResultsPanel} />
                            </Switch>
                        </Grid>
                    </Grid>
                </SplitterLayout>
            </Grid>
            <Grid item>
                <DetailsPanel />
            </Grid>
            <AdvancedTabDialog />
            <ContextMenu />
            <CopyCollectionDialog />
            <CopyProcessDialog />
            <CreateCollectionDialog />
            <CreateProjectDialog />
            <CurrentTokenDialog />
            <FileRemoveDialog />
            <FilesUploadCollectionDialog />
            <MoveCollectionDialog />
            <MoveProcessDialog />
            <MoveProjectDialog />
            <MultipleFilesRemoveDialog />
            <PartialCopyCollectionDialog />
            <ProcessCommandDialog />
            <RenameFileDialog />
            <RichTextEditorDialog />
            <SharingDialog />
            <Snackbar />
            <UpdateCollectionDialog />
            <UpdateProcessDialog />
            <UpdateProjectDialog />
        </Grid>
    );
