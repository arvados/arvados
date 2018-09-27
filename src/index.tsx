// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import { MainPanel } from './views/main-panel/main-panel';
import './index.css';
import { Route } from 'react-router';
import createBrowserHistory from "history/createBrowserHistory";
import { History } from "history";
import { configureStore, RootStore } from './store/store';
import { ConnectedRouter } from "react-router-redux";
import { ApiToken } from "./views-components/api-token/api-token";
import { initAuth } from "./store/auth/auth-action";
import { createServices } from "./services/services";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from './common/custom-theme';
import { fetchConfig } from './common/config';
import { addMenuActionSet, ContextMenuKind } from './views-components/context-menu/context-menu';
import { rootProjectActionSet } from "./views-components/context-menu/action-sets/root-project-action-set";
import { projectActionSet } from "./views-components/context-menu/action-sets/project-action-set";
import { resourceActionSet } from './views-components/context-menu/action-sets/resource-action-set';
import { favoriteActionSet } from "./views-components/context-menu/action-sets/favorite-action-set";
import { collectionFilesActionSet } from './views-components/context-menu/action-sets/collection-files-action-set';
import { collectionFilesItemActionSet } from './views-components/context-menu/action-sets/collection-files-item-action-set';
import { collectionActionSet } from './views-components/context-menu/action-sets/collection-action-set';
import { collectionResourceActionSet } from './views-components/context-menu/action-sets/collection-resource-action-set';
import { processActionSet } from './views-components/context-menu/action-sets/process-action-set';
import { loadWorkbench } from './store/workbench/workbench-actions';
import { Routes } from '~/routes/routes';
import { trashActionSet } from "~/views-components/context-menu/action-sets/trash-action-set";
import { ServiceRepository } from '~/services/services';
import { initWebSocket } from '~/websocket/websocket';
import { Config } from '~/common/config';
import { addRouteChangeHandlers } from './routes/route-change-handlers';
import { setCurrentTokenDialogApiHost } from '~/store/current-token-dialog/current-token-dialog-actions';
import { processResourceActionSet } from '~/views-components/context-menu/action-sets/process-resource-action-set';
import { progressIndicatorActions } from '~/store/progress-indicator/progress-indicator-actions';
import { setUuidPrefix } from '~/store/workflow-panel/workflow-panel-actions';
import { trashedCollectionActionSet } from '~/views-components/context-menu/action-sets/trashed-collection-action-set';
import { ContainerRequestState } from '~/models/container-request';
import { MountKind } from './models/mount-types';

const getBuildNumber = () => "BN-" + (process.env.REACT_APP_BUILD_NUMBER || "dev");
const getGitCommit = () => "GIT-" + (process.env.REACT_APP_GIT_COMMIT || "latest").substr(0, 7);
const getBuildInfo = () => getBuildNumber() + " / " + getGitCommit();

const buildInfo = getBuildInfo();

console.log(`Starting arvados [${buildInfo}]`);

addMenuActionSet(ContextMenuKind.ROOT_PROJECT, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.PROJECT, projectActionSet);
addMenuActionSet(ContextMenuKind.RESOURCE, resourceActionSet);
addMenuActionSet(ContextMenuKind.FAVORITE, favoriteActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES, collectionFilesActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES_ITEM, collectionFilesItemActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION, collectionActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_RESOURCE, collectionResourceActionSet);
addMenuActionSet(ContextMenuKind.TRASHED_COLLECTION, trashedCollectionActionSet);
addMenuActionSet(ContextMenuKind.PROCESS, processActionSet);
addMenuActionSet(ContextMenuKind.PROCESS_RESOURCE, processResourceActionSet);
addMenuActionSet(ContextMenuKind.TRASH, trashActionSet);

fetchConfig()
    .then(({ config, apiHost }) => {
        const history = createBrowserHistory();
        const services = createServices(config, {
            progressFn: (id, working) => {
                store.dispatch(progressIndicatorActions.TOGGLE_WORKING({ id, working }));
            },
            errorFn: (id, error) => {
                // console.error("Backend error:", error);
                // store.dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Backend error", kind: SnackbarKind.ERROR }));
            }
        });
        const store = configureStore(history, services);

        store.subscribe(initListener(history, store, services, config));
        store.dispatch(initAuth());
        store.dispatch(setCurrentTokenDialogApiHost(apiHost));
        store.dispatch(setUuidPrefix(config.uuidPrefix));

        const TokenComponent = (props: any) => <ApiToken authService={services.authService} {...props} />;
        const MainPanelComponent = (props: any) => <MainPanel buildInfo={buildInfo} {...props} />;

        const App = () =>
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <div>
                            <Route path={Routes.TOKEN} component={TokenComponent} />
                            <Route path={Routes.ROOT} component={MainPanelComponent} />
                        </div>
                    </ConnectedRouter>
                </Provider>
            </MuiThemeProvider>;

        ReactDOM.render(
            <App />,
            document.getElementById('root') as HTMLElement
        );
    });

const initListener = (history: History, store: RootStore, services: ServiceRepository, config: Config) => {
    let initialized = false;
    return async () => {
        const { router, auth } = store.getState();
        if (router.location && auth.user && !initialized) {
            initialized = true;
            initWebSocket(config, services.authService, store);
            await store.dispatch(loadWorkbench());
            addRouteChangeHandlers(history, store);
            // createSampleProcess(services);
        }
    };
};


const createSampleProcess = ({ containerRequestService }: ServiceRepository) => {
    containerRequestService.create({
        ownerUuid: 'c97qk-j7d0g-s3ngc1z0748hsmf',
        name: 'Simple process 7',
        state: ContainerRequestState.COMMITTED,
        mounts: {
            '/var/spool/cwl': {
                kind: MountKind.COLLECTION,
                writable: true,
            },
            'stdout': {
                kind: MountKind.MOUNTED_FILE,
                path: '/var/spool/cwl/cwl.output.json'
            },
            '/var/lib/cwl/workflow.json': {
                kind: MountKind.JSON,
                content: {
                    "cwlVersion": "v1.0",
                    "$graph": [
                        {
                            "class": "CommandLineTool",
                            "requirements": [
                                {
                                  "listing": [
                                    {
                                      "entryname": "input_collector.log",
                                      "entry": "$(inputs.single_file.basename)\n"
                                    }
                                  ],
                                  "class": "InitialWorkDirRequirement"
                                }
                              ],
                            "inputs": [
                                {
                                    "type": "File",
                                    "id": "#input_collector.cwl/single_file"
                                }
                            ],
                            "outputs": [
                                {
                                    "type": "File",
                                    "outputBinding": {
                                        "glob": "*"
                                    },
                                    "id": "#input_collector.cwl/output"
                                }
                            ],
                            "baseCommand": [
                                "echo"
                            ],
                            "id": "#input_collector.cwl"
                        },
                        {
                            "class": "Workflow",
                            "doc": "This is the description of the workflow",
                            "inputs": [
                                {
                                    "type": "File",
                                    "label": "Single File",
                                    "doc": "This should allow for single File selection only.",
                                    "id": "#main/single_file"
                                }
                            ],
                            "outputs": [
                                {
                                    "type": "File",
                                    "outputSource": "#main/input_collector/output",
                                    "id": "#main/log_file"
                                }
                            ],
                            "steps": [
                                {
                                    "run": "#input_collector.cwl",
                                    "in": [
                                        {
                                            "source": "#main/single_file",
                                            "id": "#main/input_collector/single_file"
                                        }
                                    ],
                                    "out": [
                                        "#main/input_collector/output"
                                    ],
                                    "id": "#main/input_collector"
                                }
                            ],
                            "id": "#main"
                        }
                    ]
                },
            },
            '/var/lib/cwl/cwl.input.json': {
                kind: MountKind.JSON,
                content: {
                    "single_file": {
                        "class": "File",
                        "location": "keep:233454526794c0a2d56a305baeff3d30+145/1.txt",
                        "basename": "fileA"
                      }
                },
            }
        },
        runtimeConstraints: {
            API: true,
            vcpus: 1,
            ram: 1073741824,
        },
        containerImage: 'arvados/jobs:1.1.4.20180618144723',
        cwd: '/var/spool/cwl',
        command: [
            'arvados-cwl-runner',
            '--local',
            '--api=containers',
            "--project-uuid=c97qk-j7d0g-s3ngc1z0748hsmf",
            '/var/lib/cwl/workflow.json#main',
            '/var/lib/cwl/cwl.input.json'
        ],
        outputPath: '/var/spool/cwl',
        priority: 1,
    });
};

