// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { Route, Switch } from "react-router";
import { ProjectPanel } from "views/project-panel/project-panel";
import { DetailsPanel } from "views-components/details-panel/details-panel";
import { ArvadosTheme } from "common/custom-theme";
import { ContextMenu } from "views-components/context-menu/context-menu";
import { FavoritePanel } from "../favorite-panel/favorite-panel";
import { TokenDialog } from "views-components/token-dialog/token-dialog";
import { RichTextEditorDialog } from "views-components/rich-text-editor-dialog/rich-text-editor-dialog";
import { Snackbar } from "views-components/snackbar/snackbar";
import { CollectionPanel } from "../collection-panel/collection-panel";
import { RenameFileDialog } from "views-components/rename-file-dialog/rename-file-dialog";
import { FileRemoveDialog } from "views-components/file-remove-dialog/file-remove-dialog";
import { MultipleFilesRemoveDialog } from "views-components/file-remove-dialog/multiple-files-remove-dialog";
import { Routes } from "routes/routes";
import { SidePanel } from "views-components/side-panel/side-panel";
import { ProcessPanel } from "views/process-panel/process-panel";
import { ChangeWorkflowDialog } from "views-components/run-process-dialog/change-workflow-dialog";
import { CreateProjectDialog } from "views-components/dialog-forms/create-project-dialog";
import { CreateCollectionDialog } from "views-components/dialog-forms/create-collection-dialog";
import { CopyCollectionDialog, CopyMultiCollectionDialog } from "views-components/dialog-forms/copy-collection-dialog";
import { CopyProcessDialog } from "views-components/dialog-forms/copy-process-dialog";
import { UpdateCollectionDialog } from "views-components/dialog-forms/update-collection-dialog";
import { UpdateProcessDialog } from "views-components/dialog-forms/update-process-dialog";
import { UpdateProjectDialog } from "views-components/dialog-forms/update-project-dialog";
import { MoveProcessDialog } from "views-components/dialog-forms/move-process-dialog";
import { MoveProjectDialog } from "views-components/dialog-forms/move-project-dialog";
import { MoveCollectionDialog } from "views-components/dialog-forms/move-collection-dialog";
import { FilesUploadCollectionDialog } from "views-components/dialog-forms/files-upload-collection-dialog";
import { PartialCopyToNewCollectionDialog } from "views-components/dialog-forms/partial-copy-to-new-collection-dialog";
import { PartialCopyToExistingCollectionDialog } from "views-components/dialog-forms/partial-copy-to-existing-collection-dialog";
import { PartialCopyToSeparateCollectionsDialog } from "views-components/dialog-forms/partial-copy-to-separate-collections-dialog";
import { PartialMoveToNewCollectionDialog } from "views-components/dialog-forms/partial-move-to-new-collection-dialog";
import { PartialMoveToExistingCollectionDialog } from "views-components/dialog-forms/partial-move-to-existing-collection-dialog";
import { PartialMoveToSeparateCollectionsDialog } from "views-components/dialog-forms/partial-move-to-separate-collections-dialog";
import { RemoveProcessDialog } from "views-components/process-remove-dialog/process-remove-dialog";
import { RemoveWorkflowDialog } from "views-components/workflow-remove-dialog/workflow-remove-dialog";
import { MainContentBar } from "views-components/main-content-bar/main-content-bar";
import { Grid } from "@mui/material";
import { TrashPanel } from "views/trash-panel/trash-panel";
import { SharedWithMePanel } from "views/shared-with-me-panel/shared-with-me-panel";
import { RunProcessPanel } from "views/run-process-panel/run-process-panel";
import SplitterLayout from "react-splitter-layout";
import { WorkflowPanel } from "views/workflow-panel/workflow-panel";
import { RegisteredWorkflowPanel } from "views/workflow-panel/registered-workflow-panel";
import { SearchResultsPanel } from "views/search-results-panel/search-results-panel";
import { SshKeyPanel } from "views/ssh-key-panel/ssh-key-panel";
import { SshKeyAdminPanel } from "views/ssh-key-panel/ssh-key-admin-panel";
import { SiteManagerPanel } from "views/site-manager-panel/site-manager-panel";
import { UserProfilePanel } from "views/user-profile-panel/user-profile-panel";
import { SharingDialog } from "views-components/sharing-dialog/sharing-dialog";
import { NotFoundDialog } from "views-components/not-found-dialog/not-found-dialog";
import { AdvancedTabDialog } from "views-components/advanced-tab-dialog/advanced-tab-dialog";
import { ProcessInputDialog } from "views-components/process-input-dialog/process-input-dialog";
import { VirtualMachineUserPanel } from "views/virtual-machine-panel/virtual-machine-user-panel";
import { VirtualMachineAdminPanel } from "views/virtual-machine-panel/virtual-machine-admin-panel";
import { RepositoriesPanel } from "views/repositories-panel/repositories-panel";
import { KeepServicePanel } from "views/keep-service-panel/keep-service-panel";
import { ApiClientAuthorizationPanel } from "views/api-client-authorization-panel/api-client-authorization-panel";
import { LinkPanel } from "views/link-panel/link-panel";
import { RepositoriesSampleGitDialog } from "views-components/repositories-sample-git-dialog/repositories-sample-git-dialog";
import { RepositoryAttributesDialog } from "views-components/repository-attributes-dialog/repository-attributes-dialog";
import { CreateRepositoryDialog } from "views-components/dialog-forms/create-repository-dialog";
import { RemoveRepositoryDialog } from "views-components/repository-remove-dialog/repository-remove-dialog";
import { CreateSshKeyDialog } from "views-components/dialog-forms/create-ssh-key-dialog";
import { PublicKeyDialog } from "views-components/ssh-keys-dialog/public-key-dialog";
import { RemoveApiClientAuthorizationDialog } from "views-components/api-client-authorizations-dialog/remove-dialog";
import { RemoveKeepServiceDialog } from "views-components/keep-services-dialog/remove-dialog";
import { RemoveLinkDialog } from "views-components/links-dialog/remove-dialog";
import { RemoveSshKeyDialog } from "views-components/ssh-keys-dialog/remove-dialog";
import { VirtualMachineAttributesDialog } from "views-components/virtual-machines-dialog/attributes-dialog";
import { RemoveVirtualMachineDialog } from "views-components/virtual-machines-dialog/remove-dialog";
import { RemoveVirtualMachineLoginDialog } from "views-components/virtual-machines-dialog/remove-login-dialog";
import { VirtualMachineAddLoginDialog } from "views-components/virtual-machines-dialog/add-login-dialog";
import { AttributesApiClientAuthorizationDialog } from "views-components/api-client-authorizations-dialog/attributes-dialog";
import { AttributesKeepServiceDialog } from "views-components/keep-services-dialog/attributes-dialog";
import { AttributesLinkDialog } from "views-components/links-dialog/attributes-dialog";
import { AttributesSshKeyDialog } from "views-components/ssh-keys-dialog/attributes-dialog";
import { UserPanel } from "views/user-panel/user-panel";
import { UserAttributesDialog } from "views-components/user-dialog/attributes-dialog";
import { CreateUserDialog } from "views-components/dialog-forms/create-user-dialog";
import { HelpApiClientAuthorizationDialog } from "views-components/api-client-authorizations-dialog/help-dialog";
import { DeactivateDialog } from "views-components/user-dialog/deactivate-dialog";
import { ActivateDialog } from "views-components/user-dialog/activate-dialog";
import { SetupDialog } from "views-components/user-dialog/setup-dialog";
import { GroupsPanel } from "views/groups-panel/groups-panel";
import { RemoveGroupDialog } from "views-components/groups-dialog/remove-dialog";
import { GroupAttributesDialog } from "views-components/groups-dialog/attributes-dialog";
import { GroupDetailsPanel } from "views/group-details-panel/group-details-panel";
import { RemoveGroupMemberDialog } from "views-components/groups-dialog/member-remove-dialog";
import { GroupMemberAttributesDialog } from "views-components/groups-dialog/member-attributes-dialog";
import { PublicFavoritePanel } from "views/public-favorites-panel/public-favorites-panel";
import { LinkAccountPanel } from "views/link-account-panel/link-account-panel";
import { CollectionsContentAddressPanel } from "views/collection-content-address-panel/collection-content-address-panel";
import { AllProcessesPanel } from "../all-processes-panel/all-processes-panel";
import { NotFoundPanel } from "../not-found-panel/not-found-panel";
import { AutoLogout } from "views-components/auto-logout/auto-logout";
import { RestoreCollectionVersionDialog } from "views-components/collections-dialog/restore-version-dialog";
import { WebDavS3InfoDialog } from "views-components/webdav-s3-dialog/webdav-s3-dialog";
import { pluginConfig } from "plugins";
import { ElementListReducer } from "common/plugintypes";
import { Banner } from "views-components/baner/banner";
import { InstanceTypesPanel } from "views/instance-types-panel/instance-types-panel";
import classNames from "classnames";

type CssRules = "root" | "container" | "splitter" | "splitterSidePanel" | "splitterDetails" | "asidePanel" | "contentWrapper" | "content";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        paddingTop: theme.spacing(7),
        background: theme.palette.background.default,
    },
    container: {
        position: "relative",
    },
    splitter: {
        "& > .layout-splitter": {
            width: "3px",
        },
        "& > .layout-splitter[disabled]": {
            pointerEvents: "none",
            cursor: "pointer",
        },
        "& > .layout-pane": {
            overflow: "hidden auto",
        },
    },
    splitterSidePanel: {
        "& > .layout-splitter::after": {
            content: `""`,
            marginLeft: "3px", // Matches splitter line width
            width: "8px",
            display: "block",
            position: "relative",
            height: "100%",
            zIndex: 100, // Needed for drag handle to overlap middle panel
        }
    },
    splitterDetails: {
        "& > .layout-splitter::after": {
            content: `""`,
            marginLeft: "-8px",
            width: "8px",
            display: "block",
            position: "relative",
            height: "100%",
        }
    },
    asidePanel: {
        paddingTop: theme.spacing(1),
        height: "100%",
    },
    contentWrapper: {
        paddingTop: theme.spacing(1),
        minWidth: 0,
    },
    content: {
        minWidth: 0,
        maxWidth: "100%",
        paddingLeft: theme.spacing(3),
        paddingRight: theme.spacing(3),
        // Reserve vertical space for app bar + MainContentBar
        minHeight: `calc(100vh - ${theme.spacing(16)})`,
        display: "flex",
    },
});

let routes = (
    <>
        <Route
            path={Routes.PROJECTS}
            component={ProjectPanel}
        />
        <Route
            path={Routes.COLLECTIONS}
            component={CollectionPanel}
        />
        <Route
            path={Routes.FAVORITES}
            component={FavoritePanel}
        />
        <Route
            path={Routes.ALL_PROCESSES}
            component={AllProcessesPanel}
        />
        <Route
            path={Routes.PROCESSES}
            component={ProcessPanel}
        />
        <Route
            path={Routes.TRASH}
            component={TrashPanel}
        />
        <Route
            path={Routes.SHARED_WITH_ME}
            component={SharedWithMePanel}
        />
        <Route
            path={Routes.RUN_PROCESS}
            component={RunProcessPanel}
        />
        <Route
            path={Routes.REGISTEREDWORKFLOW}
            component={RegisteredWorkflowPanel}
        />
        <Route
            path={Routes.WORKFLOWS}
            component={WorkflowPanel}
        />
        <Route
            path={Routes.SEARCH_RESULTS}
            component={SearchResultsPanel}
        />
        <Route
            path={Routes.VIRTUAL_MACHINES_USER}
            component={VirtualMachineUserPanel}
        />
        <Route
            path={Routes.VIRTUAL_MACHINES_ADMIN}
            component={VirtualMachineAdminPanel}
        />
        <Route
            path={Routes.REPOSITORIES}
            component={RepositoriesPanel}
        />
        <Route
            path={Routes.SSH_KEYS_USER}
            component={SshKeyPanel}
        />
        <Route
            path={Routes.SSH_KEYS_ADMIN}
            component={SshKeyAdminPanel}
        />
        <Route
            path={Routes.INSTANCE_TYPES}
            component={InstanceTypesPanel}
        />
        <Route
            path={Routes.SITE_MANAGER}
            component={SiteManagerPanel}
        />
        <Route
            path={Routes.KEEP_SERVICES}
            component={KeepServicePanel}
        />
        <Route
            path={Routes.USERS}
            component={UserPanel}
        />
        <Route
            path={Routes.API_CLIENT_AUTHORIZATIONS}
            component={ApiClientAuthorizationPanel}
        />
        <Route
            path={Routes.MY_ACCOUNT}
            component={UserProfilePanel}
        />
        <Route
            path={Routes.USER_PROFILE}
            component={UserProfilePanel}
        />
        <Route
            path={Routes.GROUPS}
            component={GroupsPanel}
        />
        <Route
            path={Routes.GROUP_DETAILS}
            component={GroupDetailsPanel}
        />
        <Route
            path={Routes.LINKS}
            component={LinkPanel}
        />
        <Route
            path={Routes.PUBLIC_FAVORITES}
            component={PublicFavoritePanel}
        />
        <Route
            path={Routes.LINK_ACCOUNT}
            component={LinkAccountPanel}
        />
        <Route
            path={Routes.COLLECTIONS_CONTENT_ADDRESS}
            component={CollectionsContentAddressPanel}
        />
    </>
);

const reduceRoutesFn: (a: React.ReactElement[], b: ElementListReducer) => React.ReactElement[] = (a, b) => b(a);

routes = React.createElement(
    React.Fragment,
    null,
    pluginConfig.centerPanelList.reduce(reduceRoutesFn, React.Children.toArray(routes.props.children))
);

type SplitterPanelSettings = {
    storageKey: string;
    minSize: number;
    defaultSize: number;
}

interface WorkbenchDataProps {
    isUserActive: boolean;
    isNotLinking: boolean;
    sessionIdleTimeout: number;
    sidePanelIsCollapsed: boolean;
    isDetailsPanelOpen: boolean;
}

type WorkbenchPanelProps = WithStyles<CssRules> & WorkbenchDataProps;

export const WorkbenchPanel = withStyles(styles)((props: WorkbenchPanelProps) => {
    const { classes, sidePanelIsCollapsed, isNotLinking, isDetailsPanelOpen, isUserActive, sessionIdleTimeout } = props;

    const SIDE_PANEL_COLLAPSED_WIDTH = 50;
    const MAIN_PANEL_MIN_SIZE = 300;

    const splitterSettings: Record<string, SplitterPanelSettings> = {
        LEFT: {
            storageKey: "splitterSize",
            minSize: 210,
            defaultSize: 240,
        },
        RIGHT: {
            storageKey: "detailsPanelSplitterSize",
            minSize: 250,
            defaultSize: 320,
        },
    };

    const saveSplitterSize = (panel: SplitterPanelSettings) => (size: number) => {
        localStorage.setItem(panel.storageKey, size.toString());
        if (panel.storageKey === splitterSettings.LEFT.storageKey) {
            // Trigger resize on subSplitters when LEFT panel resized
            nestedSplitter.current && nestedSplitter.current.handleResize();
        }
    };

    const getSplitterInitialSize = (panel: SplitterPanelSettings) => {
        const storedSize = localStorage.getItem(panel.storageKey);
        return storedSize ? Math.max(Number(storedSize), panel.minSize) : panel.defaultSize;
    };

    // Updates left panel collapsed state
    const applyCollapsedState = () => {
        const sidePanel: Element = document.getElementsByClassName("layout-pane")[0];

        if (sidePanel) {
            if (sidePanelIsCollapsed) {
                // Using max-width overrides any resize calculations when left panel is collapsed
                sidePanel.setAttribute("style", `max-width: ${SIDE_PANEL_COLLAPSED_WIDTH}px`);
            } else {
                sidePanel.setAttribute("style", `width: ${getSplitterInitialSize(splitterSettings.LEFT)}px`);
            }
        }

        const splitter = document.getElementsByClassName("layout-splitter")[0];
        sidePanelIsCollapsed ? splitter?.setAttribute("disabled", "") : splitter?.removeAttribute("disabled");

        // Trigger resize on subSplitters
        nestedSplitter.current && nestedSplitter.current.handleResize();
    };

    const nestedSplitter = React.useRef<{ handleResize: () => void }>();

    applyCollapsedState();

    return (
        <Grid
            container
            item
            xs
            className={classes.root}
        >
            {sessionIdleTimeout > 0 && <AutoLogout />}
            <Grid
                container
                item
                xs
                className={classes.container}
            >
                <SplitterLayout
                    customClassName={classNames(classes.splitter, classes.splitterSidePanel)}
                    percentage={false}
                    primaryIndex={1}
                    secondaryInitialSize={getSplitterInitialSize(splitterSettings.LEFT)}
                    secondaryMinSize={splitterSettings.LEFT.minSize}
                    primaryMinSize={MAIN_PANEL_MIN_SIZE}
                    // Resize event only exists for secondary
                    onSecondaryPaneSizeChange={saveSplitterSize(splitterSettings.LEFT)}
                >
                    {isUserActive && isNotLinking && (
                        <Grid
                            container
                            item
                            xs
                            component="aside"
                            direction="column"
                            className={classes.asidePanel}
                        >
                            <SidePanel />
                        </Grid>
                    )}
                    <Grid
                        container
                        item
                        xs
                    >
                        <SplitterLayout
                            customClassName={classNames(classes.splitter, classes.splitterDetails)}
                            percentage={false}
                            primaryIndex={0}
                            primaryMinSize={MAIN_PANEL_MIN_SIZE}
                            secondaryInitialSize={getSplitterInitialSize(splitterSettings.RIGHT)}
                            secondaryMinSize={splitterSettings.RIGHT.minSize}
                            onSecondaryPaneSizeChange={saveSplitterSize(splitterSettings.RIGHT)}
                            ref={nestedSplitter}
                        >
                            <Grid
                                container
                                item
                                xs
                                component="main"
                                direction="column"
                                className={classes.contentWrapper}
                            >
                                <Grid xs>
                                    {isNotLinking && <MainContentBar />}
                                </Grid>
                                <Grid
                                    className={classes.content}
                                >
                                    <Switch>
                                        {routes.props.children}
                                        <Route
                                            path={Routes.NO_MATCH}
                                            component={NotFoundPanel}
                                        />
                                    </Switch>
                                </Grid>
                            </Grid>
                            {isDetailsPanelOpen && <Grid item style={{height: "100%"}}>
                                <DetailsPanel />
                            </Grid>}
                        </SplitterLayout>
                    </Grid>
                </SplitterLayout>
            </Grid>
            <AdvancedTabDialog />
            <AttributesApiClientAuthorizationDialog />
            <AttributesKeepServiceDialog />
            <AttributesLinkDialog />
            <AttributesSshKeyDialog />
            <ChangeWorkflowDialog />
            <ContextMenu />
            <CopyCollectionDialog />
            <CopyMultiCollectionDialog />
            <CopyProcessDialog />
            <CreateCollectionDialog />
            <CreateProjectDialog />
            <CreateRepositoryDialog />
            <CreateSshKeyDialog />
            <CreateUserDialog />
            <TokenDialog />
            <FileRemoveDialog />
            <FilesUploadCollectionDialog />
            <GroupAttributesDialog />
            <GroupMemberAttributesDialog />
            <HelpApiClientAuthorizationDialog />
            <MoveCollectionDialog />
            <MoveProcessDialog />
            <MoveProjectDialog />
            <MultipleFilesRemoveDialog />
            <PublicKeyDialog />
            <PartialCopyToNewCollectionDialog />
            <PartialCopyToExistingCollectionDialog />
            <PartialCopyToSeparateCollectionsDialog />
            <PartialMoveToNewCollectionDialog />
            <PartialMoveToExistingCollectionDialog />
            <PartialMoveToSeparateCollectionsDialog />
            <ProcessInputDialog />
            <RestoreCollectionVersionDialog />
            <RemoveApiClientAuthorizationDialog />
            <RemoveGroupDialog />
            <RemoveGroupMemberDialog />
            <RemoveKeepServiceDialog />
            <RemoveLinkDialog />
            <RemoveProcessDialog />
            <RemoveWorkflowDialog />
            <RemoveRepositoryDialog />
            <RemoveSshKeyDialog />
            <RemoveVirtualMachineDialog />
            <RemoveVirtualMachineLoginDialog />
            <VirtualMachineAddLoginDialog />
            <RenameFileDialog />
            <RepositoryAttributesDialog />
            <RepositoriesSampleGitDialog />
            <RichTextEditorDialog />
            <SharingDialog />
            <NotFoundDialog />
            <Snackbar />
            <UpdateCollectionDialog />
            <UpdateProcessDialog />
            <UpdateProjectDialog />
            <UserAttributesDialog />
            <DeactivateDialog />
            <ActivateDialog />
            <SetupDialog />
            <VirtualMachineAttributesDialog />
            <WebDavS3InfoDialog />
            <Banner />
            {React.createElement(React.Fragment, null, pluginConfig.dialogs)}
        </Grid>
    );
});
