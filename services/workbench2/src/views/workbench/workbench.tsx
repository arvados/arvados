// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from "react";
import { StyleRulesCallback, WithStyles, withStyles } from "@material-ui/core/styles";
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
import { Grid } from "@material-ui/core";
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
import { FedLogin } from "./fed-login";
import { CollectionsContentAddressPanel } from "views/collection-content-address-panel/collection-content-address-panel";
import { AllProcessesPanel } from "../all-processes-panel/all-processes-panel";
import { NotFoundPanel } from "../not-found-panel/not-found-panel";
import { AutoLogout } from "views-components/auto-logout/auto-logout";
import { RestoreCollectionVersionDialog } from "views-components/collections-dialog/restore-version-dialog";
import { WebDavS3InfoDialog } from "views-components/webdav-s3-dialog/webdav-s3-dialog";
import { pluginConfig } from "plugins";
import { ElementListReducer } from "common/plugintypes";
import { COLLAPSE_ICON_SIZE } from "views-components/side-panel-toggle/side-panel-toggle";
import { Banner } from "views-components/baner/banner";
import { InstanceTypesPanel } from "views/instance-types-panel/instance-types-panel";

type CssRules = "root" | "container" | "splitter" | "asidePanel" | "contentWrapper" | "content";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        paddingTop: theme.spacing.unit * 7,
        background: theme.palette.background.default,
    },
    container: {
        position: "relative",
    },
    splitter: {
        "& > .layout-splitter": {
            width: "3px",
        },
        "& > .layout-splitter-disabled": {
            pointerEvents: "none",
            cursor: "pointer",
        },
    },
    asidePanel: {
        paddingTop: theme.spacing.unit,
        height: "100%",
    },
    contentWrapper: {
        paddingTop: theme.spacing.unit,
        minWidth: 0,
    },
    content: {
        minWidth: 0,
        paddingLeft: theme.spacing.unit * 3,
        paddingRight: theme.spacing.unit * 3,
        // Reserve vertical space for app bar + MainContentBar
        minHeight: `calc(100vh - ${theme.spacing.unit * 16}px)`,
        display: "flex",
    },
});

interface WorkbenchDataProps {
    isUserActive: boolean;
    isNotLinking: boolean;
    sessionIdleTimeout: number;
    sidePanelIsCollapsed: boolean;
    isTransitioning: boolean;
    currentSideWidth: number;
}

type WorkbenchPanelProps = WithStyles<CssRules> & WorkbenchDataProps;

const defaultSplitterSize = 90;

const getSplitterInitialSize = () => {
    const splitterSize = localStorage.getItem("splitterSize");
    return splitterSize ? Number(splitterSize) : defaultSplitterSize;
};

const saveSplitterSize = (size: number) => localStorage.setItem("splitterSize", size.toString());

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

export const WorkbenchPanel = withStyles(styles)((props: WorkbenchPanelProps) => {
const { classes, sidePanelIsCollapsed, isNotLinking, isTransitioning, isUserActive, sessionIdleTimeout, currentSideWidth } = props

    const applyCollapsedState = (savedWidthInPx) => {
        const rightPanel: Element = document.getElementsByClassName("layout-pane")[1];
        const totalWidth: number = document.getElementsByClassName("splitter-layout")[0]?.clientWidth;
        const savedWidthInPercent = (savedWidthInPx / totalWidth) * 100
        const rightPanelExpandedWidth = (totalWidth - COLLAPSE_ICON_SIZE) / (totalWidth / 100);

        if(isTransitioning && !!rightPanel) {
            rightPanel.setAttribute('style', `width: ${sidePanelIsCollapsed ? `calc(${savedWidthInPercent}% - 1rem)` : `${getSplitterInitialSize()}%`};`)
        }

        if (rightPanel) {
            rightPanel.setAttribute("style", `width: ${sidePanelIsCollapsed ? `calc(${rightPanelExpandedWidth}% - 1rem)` : `${getSplitterInitialSize()}%`};`);
        }
        const splitter = document.getElementsByClassName("layout-splitter")[0];
        sidePanelIsCollapsed ? splitter?.classList.add("layout-splitter-disabled") : splitter?.classList.remove("layout-splitter-disabled");
    };

    const [savedWidth, setSavedWidth] = useState<number>(0)

    useEffect(()=>{
        if (isTransitioning) setSavedWidth(currentSideWidth)
    }, [isTransitioning, currentSideWidth])

    useEffect(()=>{
        if (isTransitioning) applyCollapsedState(savedWidth);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isTransitioning, savedWidth])

    applyCollapsedState(savedWidth);

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
                    customClassName={classes.splitter}
                    percentage={true}
                    primaryIndex={0}
                    primaryMinSize={10}
                    secondaryInitialSize={getSplitterInitialSize()}
                    secondaryMinSize={40}
                    onSecondaryPaneSizeChange={saveSplitterSize}
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
                        component="main"
                        direction="column"
                        className={classes.contentWrapper}
                    >
                        <Grid
                            item
                            xs
                        >
                            {isNotLinking && <MainContentBar />}
                        </Grid>
                        <Grid
                            item
                            xs
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
                </SplitterLayout>
            </Grid>
            <Grid item>
                <DetailsPanel />
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
            <FedLogin />
            <WebDavS3InfoDialog />
            <Banner />
            {React.createElement(React.Fragment, null, pluginConfig.dialogs)}
        </Grid>
    );
});
