// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import { MainPanel } from './views/main-panel/main-panel';
import './index.css';
import { Route, Switch } from 'react-router';
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
import { MountKind } from '~/models/mount-types';
import { setBuildInfo } from '~/store/app-info/app-info-actions';
import { getBuildInfo } from '~/common/app-info';
import { DragDropContextProvider } from 'react-dnd';
import HTML5Backend from 'react-dnd-html5-backend';
import { initAdvanceFormProjectsTree } from '~/store/search-bar/search-bar-actions';

console.log(`Starting arvados [${getBuildInfo()}]`);

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
        store.dispatch(setBuildInfo());
        store.dispatch(setCurrentTokenDialogApiHost(apiHost));
        store.dispatch(setUuidPrefix(config.uuidPrefix));

        const TokenComponent = (props: any) => <ApiToken authService={services.authService} {...props} />;
        const MainPanelComponent = (props: any) => <MainPanel {...props} />;

        const App = () =>
            <MuiThemeProvider theme={CustomTheme}>
                <DragDropContextProvider backend={HTML5Backend}>
                    <Provider store={store}>
                        <ConnectedRouter history={history}>
                            <Switch>
                                <Route path={Routes.TOKEN} component={TokenComponent} />
                                <Route path={Routes.ROOT} component={MainPanelComponent} />
                            </Switch>
                        </ConnectedRouter>
                    </Provider>
                </DragDropContextProvider>
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
            // ToDo: move to searchBar component
            store.dispatch(initAdvanceFormProjectsTree());
        }
    };
};

const createDirectoriesArrayCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'Directories array collector',
        description: 'Workflow for collecting directories array',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"multiple_collections\":\n          $(inputs.multiple_collections)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type:\n      type: array\n      items: Directory\n    id: '#input_collector.cwl/multiple_collections'\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type:\n      type: array\n      items: Directory\n    label: Multiple Collections\n    doc: This should allow for selecting multiple collections.\n    id: '#main/multiple_collections'\n    default:\n    - class: Directory\n      location: keep:1e1682585d576f031b2d8b4944f989ee+57\n      basename: 1e1682585d576f031b2d8b4944f989ee+57\n    - class: Directory\n      location: keep:326f692370e9e121fcbd013796f7352a+57\n      basename: 326f692370e9e121fcbd013796f7352a+57\n  \n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/multiple_collections'\n      id: '#main/input_collector/multiple_collections'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createPrimitiveArraysCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'String, Int and Float arrays collector',
        description: 'Workflow for collecting primitive data arrays',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"string array\":\n          $(inputs.example_string_array)\n        \"int array\":\n          $(inputs.example_int_array)\n        \"float array\":\n          $(inputs.example_float_array)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type:\n      type: array\n      items: string\n    id: '#input_collector.cwl/example_string_array'\n  - type:\n      type: array\n      items: int\n    id: '#input_collector.cwl/example_int_array'\n  - type:\n      type: array\n      items: float\n    id: '#input_collector.cwl/example_float_array'\n  \n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type:\n      type: array\n      items: string\n    label: Freetext Array\n    doc: This should allow for entering multiple strings.\n    id: '#main/example_string_array'\n    default:\n    - This is the first string\n    - This is the second string\n  - type:\n      type: array\n      items: int\n    label: Integer Array\n    doc: This should allow for entering multiple integers.\n    id: '#main/example_int_array'\n    default:\n    - 3\n    - 6\n  - type:\n      type: array\n      items: float\n    label: Float Array\n    doc: This should allow for entering multiple floats.\n    id: '#main/example_float_array'\n    default:\n    - 3.33\n    - 66.6\n\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/example_string_array'\n      id: '#main/input_collector/example_string_array'\n    - source: '#main/example_int_array'\n      id: '#main/input_collector/example_int_array'\n    - source: '#main/example_float_array'\n      id: '#main/input_collector/example_float_array'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createFilesArrayCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'Files array collector',
        description: 'Workflow for collecting files array',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"multiple_files\":\n          $(inputs.multiple_files)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type:\n      type: array\n      items: File\n    id: '#input_collector.cwl/multiple_files'\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n\n  baseCommand: [cat]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type:\n      type: array\n      items: File\n    label: Multiple Files\n    doc: This should allow for selecting multiple files.\n    id: '#main/multiple_files'\n    default:\n      - class: File\n        location: keep:af831660d820bcbb98f473355e6e1b85+67/fileA\n        basename: fileA\n        nameroot: fileA\n        nameext: ''\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/multiple_files'\n      id: '#main/input_collector/multiple_files'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createPrimitivesCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'Primitive values collector',
        description: 'Workflow for collecting primitive values',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"flag\":\n          $(inputs.example_flag)\n        \"string\":\n          $(inputs.example_string)\n        \"int\":\n          $(inputs.example_int)\n        \"long\":\n          $(inputs.example_long)\n        \"float\":\n          $(inputs.example_float)\n        \"double\":\n          $(inputs.example_double)\n    class: InitialWorkDirRequirement\n  inputs:\n  - type: double\n    id: '#input_collector.cwl/example_double'\n  - type: boolean\n    id: '#input_collector.cwl/example_flag'\n  - type: float\n    id: '#input_collector.cwl/example_float'\n  - type: int\n    id: '#input_collector.cwl/example_int'\n  - type: long\n    id: '#input_collector.cwl/example_long'\n  - type: string\n    id: '#input_collector.cwl/example_string'\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: Workflw for collecting primitive values\n  inputs:\n  - type: double\n    label: Double value\n    doc: This should allow for entering a decimal number (64-bit).\n    id: '#main/example_double'\n    default: 0.3333333333333333\n  - type: boolean\n    label: Boolean Flag\n    doc: This should render as in checkbox.\n    id: '#main/example_flag'\n    default: true\n  - type: float\n    label: Float value\n    doc: This should allow for entering a decimal number (32-bit).\n    id: '#main/example_float'\n    default: 0.15625\n  - type: int\n    label: Integer Number\n    doc: This should allow for entering a number (32-bit signed).\n    id: '#main/example_int'\n    default: 2147483647\n  - type: long\n    label: Long Number\n    doc: This should allow for entering a number (64-bit signed).\n    id: '#main/example_long'\n    default: 9223372036854775807\n  - type: string\n    label: Freetext\n    doc: This should allow for entering an arbitrary char sequence.\n    id: '#main/example_string'\n    default: This is a string\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/example_double'\n      id: '#main/input_collector/example_double'\n    - source: '#main/example_flag'\n      id: '#main/input_collector/example_flag'\n    - source: '#main/example_float'\n      id: '#main/input_collector/example_float'\n    - source: '#main/example_int'\n      id: '#main/input_collector/example_int'\n    - source: '#main/example_long'\n      id: '#main/input_collector/example_long'\n    - source: '#main/example_string'\n      id: '#main/input_collector/example_string'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createEnumCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'Enum values collector',
        description: 'Workflow for collecting enum values',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"enum_type\":\n          $(inputs.enum_type)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type:\n      type: enum\n      symbols: ['#input_collector.cwl/enum_type/OTU table', '#input_collector.cwl/enum_type/Pathway\n          table', '#input_collector.cwl/enum_type/Function table', '#input_collector.cwl/enum_type/Ortholog\n          table']\n    id: '#input_collector.cwl/enum_type'\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type:\n      type: enum\n      symbols: ['#main/enum_type/OTU table', '#main/enum_type/Pathway table', '#main/enum_type/Function\n          table', '#main/enum_type/Ortholog table']\n      name: '#enum_typef4179c7f-45f9-482d-a5db-1abb86698384'\n    label: Enumeration Type\n    doc: This should render as a drop-down menu.\n    id: '#main/enum_type'\n    default: OTU table\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/enum_type'\n      id: '#main/input_collector/enum_type'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createFilesCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'File values collector',
        description: 'Workflow for collecting file values',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"single_file\":\n          $(inputs.single_file.basename)\n        \"optional_file\":\n          $(inputs.optional_file.basename)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type:\n    - 'null'\n    - File\n    id: '#input_collector.cwl/optional_file'\n  - type:\n    - 'null'\n    - File\n    id: '#input_collector.cwl/optional_file_missing_label'\n  - type: File\n    id: '#input_collector.cwl/single_file'\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type:\n    - 'null'\n    - File\n    label: Single File (Optional)\n    doc: This should allow for single File selection only. Input should be marked\n      as optional and not enforced by form validation.\n    id: '#main/optional_file'\n    default:\n      class: File\n      location: keep:af831660d820bcbb98f473355e6e1b85+67/fileA\n      basename: fileA\n      nameroot: fileA\n      nameext: ''\n  - type:\n    - 'null'\n    - File\n    doc: Label should be the input field name because of missing label.\n    id: '#main/optional_file_missing_label'\n  - type: File\n    label: Single File\n    doc: This should allow for single File selection only.\n    id: '#main/single_file'\n    default:\n      class: File\n      location: keep:af831660d820bcbb98f473355e6e1b85+67/fileA\n      basename: fileA\n      nameroot: fileA\n      nameext: ''\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/optional_file'\n      id: '#main/input_collector/optional_file'\n    - source: '#main/single_file'\n      id: '#main/input_collector/single_file'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
};

const createCollectionCollectorWorkflow = ({ workflowService }: ServiceRepository) => {
    workflowService.create({
        name: 'Collection value collector',
        description: 'Workflow for collecting a collecion',
        definition: "cwlVersion: v1.0\n$graph:\n- class: CommandLineTool\n\n  requirements:\n  - listing:\n    - entryname: input_collector.log\n      entry: |\n        \"collection\":\n          $(inputs.collection.location)\n\n    class: InitialWorkDirRequirement\n  inputs:\n  - type: Directory\n    id: '#input_collector.cwl/collection'\n\n  outputs:\n  - type: File\n    outputBinding:\n      glob: '*'\n    id: '#input_collector.cwl/output'\n\n  baseCommand: [echo]\n  id: '#input_collector.cwl'\n- class: Workflow\n  doc: This is the description of the workflow\n  inputs:\n  - type: Directory\n    label: Single Collection\n    doc: This should allow for single Collection selection only.\n    id: '#main/collection'\n    default:\n      class: Directory\n      location: keep:af831660d820bcbb98f473355e6e1b85+67\n      basename: af831660d820bcbb98f473355e6e1b85+67\n  outputs:\n  - type: File\n    outputSource: '#main/input_collector/output'\n\n    id: '#main/log_file'\n  steps:\n  - run: '#input_collector.cwl'\n    in:\n    - source: '#main/collection'\n      id: '#main/input_collector/collection'\n    out: ['#main/input_collector/output']\n    id: '#main/input_collector'\n  id: '#main'\n",
    });
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

