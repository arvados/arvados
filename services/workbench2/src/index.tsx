// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import ReactDOM from "react-dom";
import { Provider } from "react-redux";
import { MainPanel } from "views/main-panel/main-panel";
import "index.css";
import { Route, Switch } from "react-router";
import { createBrowserHistory } from "history";
import { History } from "history";
import { configureStore, RootStore } from "store/store";
import { ConnectedRouter } from "react-router-redux";
import { ApiToken } from "views-components/api-token/api-token";
import { AddSession } from "views-components/add-session/add-session";
import { initAuth, logout } from "store/auth/auth-action";
import { createServices } from "services/services";
import { MuiThemeProvider } from "@material-ui/core/styles";
import { CustomTheme } from "common/custom-theme";
import { fetchConfig } from "common/config";
import servicesProvider from "common/service-provider";
import { addMenuActionSet } from "views-components/context-menu/context-menu";
import { ContextMenuKind } from "views-components/context-menu/menu-item-sort";
import { rootProjectActionSet } from "views-components/context-menu/action-sets/root-project-action-set";
import {
    filterGroupActionSet,
    frozenActionSet,
    projectActionSet,
    readOnlyProjectActionSet,
} from "views-components/context-menu/action-sets/project-action-set";
import { resourceActionSet } from "views-components/context-menu/action-sets/resource-action-set";
import { favoriteActionSet } from "views-components/context-menu/action-sets/favorite-action-set";
import {
    collectionFilesActionSet,
    collectionFilesMultipleActionSet,
    readOnlyCollectionFilesActionSet,
    readOnlyCollectionFilesMultipleActionSet,
} from "views-components/context-menu/action-sets/collection-files-action-set";
import {
    collectionDirectoryItemActionSet,
    collectionFileItemActionSet,
    readOnlyCollectionDirectoryItemActionSet,
    readOnlyCollectionFileItemActionSet,
} from "views-components/context-menu/action-sets/collection-files-item-action-set";
import { collectionFilesNotSelectedActionSet } from "views-components/context-menu/action-sets/collection-files-not-selected-action-set";
import {
    collectionActionSet,
    collectionAdminActionSet,
    oldCollectionVersionActionSet,
    readOnlyCollectionActionSet,
} from "views-components/context-menu/action-sets/collection-action-set";
import { loadWorkbench } from "store/workbench/workbench-actions";
import { Routes } from "routes/routes";
import { trashActionSet } from "views-components/context-menu/action-sets/trash-action-set";
import { ServiceRepository } from "services/services";
import { initWebSocket } from "websocket/websocket";
import { Config } from "common/config";
import { addRouteChangeHandlers } from "./routes/route-change-handlers";
import { setTokenDialogApiHost } from "store/token-dialog/token-dialog-actions";
import {
    processResourceActionSet,
    runningProcessResourceActionSet,
    processResourceAdminActionSet,
    runningProcessResourceAdminActionSet,
    readOnlyProcessResourceActionSet,
} from "views-components/context-menu/action-sets/process-resource-action-set";
import { trashedCollectionActionSet } from "views-components/context-menu/action-sets/trashed-collection-action-set";
import { setBuildInfo } from "store/app-info/app-info-actions";
import { getBuildInfo } from "common/app-info";
import { DragDropContextProvider } from "react-dnd";
import HTML5Backend from "react-dnd-html5-backend";
import { initAdvancedFormProjectsTree } from "store/search-bar/search-bar-actions";
import { repositoryActionSet } from "views-components/context-menu/action-sets/repository-action-set";
import { sshKeyActionSet } from "views-components/context-menu/action-sets/ssh-key-action-set";
import { keepServiceActionSet } from "views-components/context-menu/action-sets/keep-service-action-set";
import { loadVocabulary } from "store/vocabulary/vocabulary-actions";
import { virtualMachineActionSet } from "views-components/context-menu/action-sets/virtual-machine-action-set";
import { userActionSet } from "views-components/context-menu/action-sets/user-action-set";
import { UserDetailsActionSet } from "views-components/context-menu/action-sets/user-details-action-set";
import { apiClientAuthorizationActionSet } from "views-components/context-menu/action-sets/api-client-authorization-action-set";
import { groupActionSet } from "views-components/context-menu/action-sets/group-action-set";
import { groupMemberActionSet } from "views-components/context-menu/action-sets/group-member-action-set";
import { linkActionSet } from "views-components/context-menu/action-sets/link-action-set";
import { loadFileViewersConfig } from "store/file-viewers/file-viewers-actions";
import {
    filterGroupAdminActionSet,
    frozenAdminActionSet,
    projectAdminActionSet,
} from "views-components/context-menu/action-sets/project-admin-action-set";
import { permissionEditActionSet } from "views-components/context-menu/action-sets/permission-edit-action-set";
import { workflowActionSet, readOnlyWorkflowActionSet } from "views-components/context-menu/action-sets/workflow-action-set";
import { storeRedirects } from "./common/redirect-to";
import { searchResultsActionSet } from "views-components/context-menu/action-sets/search-results-action-set";

import 'bootstrap/dist/css/bootstrap.min.css';
import '@coreui/coreui/dist/css/coreui.min.css';

console.log(`Starting arvados [${getBuildInfo()}]`);

addMenuActionSet(ContextMenuKind.ROOT_PROJECT, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.PROJECT, projectActionSet);
addMenuActionSet(ContextMenuKind.READONLY_PROJECT, readOnlyProjectActionSet);
addMenuActionSet(ContextMenuKind.FILTER_GROUP, filterGroupActionSet);
addMenuActionSet(ContextMenuKind.RESOURCE, resourceActionSet);
addMenuActionSet(ContextMenuKind.FAVORITE, favoriteActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES, collectionFilesActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES_MULTIPLE, collectionFilesMultipleActionSet);
addMenuActionSet(ContextMenuKind.READONLY_COLLECTION_FILES, readOnlyCollectionFilesActionSet);
addMenuActionSet(ContextMenuKind.READONLY_COLLECTION_FILES_MULTIPLE, readOnlyCollectionFilesMultipleActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES_NOT_SELECTED, collectionFilesNotSelectedActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_DIRECTORY_ITEM, collectionDirectoryItemActionSet);
addMenuActionSet(ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM, readOnlyCollectionDirectoryItemActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILE_ITEM, collectionFileItemActionSet);
addMenuActionSet(ContextMenuKind.READONLY_COLLECTION_FILE_ITEM, readOnlyCollectionFileItemActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION, collectionActionSet);
addMenuActionSet(ContextMenuKind.READONLY_COLLECTION, readOnlyCollectionActionSet);
addMenuActionSet(ContextMenuKind.OLD_VERSION_COLLECTION, oldCollectionVersionActionSet);
addMenuActionSet(ContextMenuKind.TRASHED_COLLECTION, trashedCollectionActionSet);
addMenuActionSet(ContextMenuKind.PROCESS_RESOURCE, processResourceActionSet);
addMenuActionSet(ContextMenuKind.RUNNING_PROCESS_RESOURCE, runningProcessResourceActionSet);
addMenuActionSet(ContextMenuKind.READONLY_PROCESS_RESOURCE, readOnlyProcessResourceActionSet);
addMenuActionSet(ContextMenuKind.TRASH, trashActionSet);
addMenuActionSet(ContextMenuKind.REPOSITORY, repositoryActionSet);
addMenuActionSet(ContextMenuKind.SSH_KEY, sshKeyActionSet);
addMenuActionSet(ContextMenuKind.VIRTUAL_MACHINE, virtualMachineActionSet);
addMenuActionSet(ContextMenuKind.KEEP_SERVICE, keepServiceActionSet);
addMenuActionSet(ContextMenuKind.USER, userActionSet);
addMenuActionSet(ContextMenuKind.USER_DETAILS, UserDetailsActionSet);
addMenuActionSet(ContextMenuKind.LINK, linkActionSet);
addMenuActionSet(ContextMenuKind.API_CLIENT_AUTHORIZATION, apiClientAuthorizationActionSet);
addMenuActionSet(ContextMenuKind.GROUPS, groupActionSet);
addMenuActionSet(ContextMenuKind.GROUP_MEMBER, groupMemberActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_ADMIN, collectionAdminActionSet);
addMenuActionSet(ContextMenuKind.PROCESS_ADMIN, processResourceAdminActionSet);
addMenuActionSet(ContextMenuKind.RUNNING_PROCESS_ADMIN, runningProcessResourceAdminActionSet);
addMenuActionSet(ContextMenuKind.PROJECT_ADMIN, projectAdminActionSet);
addMenuActionSet(ContextMenuKind.FROZEN_PROJECT, frozenActionSet);
addMenuActionSet(ContextMenuKind.FROZEN_PROJECT_ADMIN, frozenAdminActionSet);
addMenuActionSet(ContextMenuKind.FILTER_GROUP_ADMIN, filterGroupAdminActionSet);
addMenuActionSet(ContextMenuKind.PERMISSION_EDIT, permissionEditActionSet);
addMenuActionSet(ContextMenuKind.READONLY_WORKFLOW, readOnlyWorkflowActionSet);
addMenuActionSet(ContextMenuKind.WORKFLOW, workflowActionSet);
addMenuActionSet(ContextMenuKind.SEARCH_RESULTS, searchResultsActionSet);

storeRedirects();

fetchConfig().then(({ config, apiHost }) => {
    const history = createBrowserHistory();

    // Provide browser's history access to Cypress to allow programmatic
    // navigation.
    if ((window as any).Cypress) {
        (window as any).appHistory = history;
    }

    const services = createServices(config, {
        progressFn: (id, working) => {
        },
        errorFn: (id, error, showSnackBar: boolean) => {
            if (showSnackBar) {
                console.error("Backend error:", error);
                if (error.status === 401 && error.errors[0].indexOf("Not logged in") > -1) {
                    // Catch auth errors when navigating and redirect to login preserving url location
                    store.dispatch(logout(false, true));
                }
            }
        },
    });

    // be sure this is initiated before the app starts
    servicesProvider.setServices(services);

    const store = configureStore(history, services, config);

    servicesProvider.setStore(store);

    store.subscribe(initListener(history, store, services, config));
    store.dispatch(initAuth(config));
    store.dispatch(setBuildInfo());
    store.dispatch(setTokenDialogApiHost(apiHost));
    store.dispatch(loadVocabulary);
    store.dispatch(loadFileViewersConfig);

    const TokenComponent = (props: any) => (
        <ApiToken
            authService={services.authService}
            config={config}
            loadMainApp={true}
            {...props}
        />
    );
    const AddSessionComponent = (props: any) => <AddSession {...props} />;
    const FedTokenComponent = (props: any) => (
        <ApiToken
            authService={services.authService}
            config={config}
            loadMainApp={false}
            {...props}
        />
    );
    const MainPanelComponent = (props: any) => <MainPanel {...props} />;

    const App = () => (
        <MuiThemeProvider theme={CustomTheme}>
            <DragDropContextProvider backend={HTML5Backend}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <Switch>
                            <Route
                                path={Routes.TOKEN}
                                component={TokenComponent}
                            />
                            <Route
                                path={Routes.FED_LOGIN}
                                component={FedTokenComponent}
                            />
                            <Route
                                path={Routes.ADD_SESSION}
                                component={AddSessionComponent}
                            />
                            <Route
                                path={Routes.ROOT}
                                component={MainPanelComponent}
                            />
                        </Switch>
                    </ConnectedRouter>
                </Provider>
            </DragDropContextProvider>
        </MuiThemeProvider>
    );

    ReactDOM.render(<App />, document.getElementById("root") as HTMLElement);
});

const initListener = (history: History, store: RootStore, services: ServiceRepository, config: Config) => {
    let initialized = false;
    return async () => {
        const { router, auth } = store.getState();
        if (router.location && auth.user && services.authService.getApiToken() && !initialized) {
            initialized = true;
            initWebSocket(config, services.authService, store);
            await store.dispatch(loadWorkbench());
            addRouteChangeHandlers(history, store);
            // ToDo: move to searchBar component
            store.dispatch(initAdvancedFormProjectsTree());
        }
    };
};
