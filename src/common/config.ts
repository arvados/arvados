// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "axios";

export const WORKBENCH_CONFIG_URL = process.env.REACT_APP_ARVADOS_CONFIG_URL || "/config.json";

interface WorkbenchConfig {
    API_HOST: string;
    VOCABULARY_URL?: string;
    FILE_VIEWERS_CONFIG_URL?: string;
}

export interface ClusterConfigJSON {
    API: {
        UnfreezeProjectRequiresAdmin: boolean
        MaxItemsPerResponse: number
    },
    ClusterID: string;
    RemoteClusters: {
        [key: string]: {
            ActivateUsers: boolean
            Host: string
            Insecure: boolean
            Proxy: boolean
            Scheme: string
        }
    };
    Mail?: {
        SupportEmailAddress: string;
    };
    Services: {
        Controller: {
            ExternalURL: string
        }
        Workbench1: {
            ExternalURL: string
        }
        Workbench2: {
            ExternalURL: string
        }
        Websocket: {
            ExternalURL: string
        }
        WebDAV: {
            ExternalURL: string
        },
        WebDAVDownload: {
            ExternalURL: string
        },
        WebShell: {
            ExternalURL: string
        }
    };
    Workbench: {
        DisableSharingURLsUI: boolean;
        ArvadosDocsite: string;
        FileViewersConfigURL: string;
        WelcomePageHTML: string;
        InactivePageHTML: string;
        SSHHelpPageHTML: string;
        SSHHelpHostSuffix: string;
        SiteName: string;
        IdleTimeout: string;
    };
    Login: {
        LoginCluster: string;
        Google: {
            Enable: boolean;
        }
        LDAP: {
            Enable: boolean;
        }
        OpenIDConnect: {
            Enable: boolean;
        }
        PAM: {
            Enable: boolean;
        }
        SSO: {
            Enable: boolean;
        }
        Test: {
            Enable: boolean;
        }
    };
    Collections: {
        ForwardSlashNameSubstitution: string;
        ManagedProperties?: {
            [key: string]: {
                Function: string,
                Value: string,
                Protected?: boolean,
            }
        },
        TrustAllContent: boolean
    };
    Volumes: {
        [key: string]: {
            StorageClasses: {
                [key: string]: boolean;
            }
        }
    };
}

export class Config {
    baseUrl!: string;
    keepWebServiceUrl!: string;
    keepWebInlineServiceUrl!: string;
    remoteHosts!: {
        [key: string]: string
    };
    rootUrl!: string;
    uuidPrefix!: string;
    websocketUrl!: string;
    workbenchUrl!: string;
    workbench2Url!: string;
    vocabularyUrl!: string;
    fileViewersConfigUrl!: string;
    loginCluster!: string;
    clusterConfig!: ClusterConfigJSON;
    apiRevision!: number;
}

export const buildConfig = (clusterConfig: ClusterConfigJSON): Config => {
    const clusterConfigJSON = removeTrailingSlashes(clusterConfig);
    const config = new Config();
    config.rootUrl = clusterConfigJSON.Services.Controller.ExternalURL;
    config.baseUrl = `${config.rootUrl}/${ARVADOS_API_PATH}`;
    config.uuidPrefix = clusterConfigJSON.ClusterID;
    config.websocketUrl = clusterConfigJSON.Services.Websocket.ExternalURL;
    config.workbench2Url = clusterConfigJSON.Services.Workbench2.ExternalURL;
    config.workbenchUrl = clusterConfigJSON.Services.Workbench1.ExternalURL;
    config.keepWebServiceUrl = clusterConfigJSON.Services.WebDAVDownload.ExternalURL;
    config.keepWebInlineServiceUrl = clusterConfigJSON.Services.WebDAV.ExternalURL;
    config.loginCluster = clusterConfigJSON.Login.LoginCluster;
    config.clusterConfig = clusterConfigJSON;
    config.apiRevision = 0;
    mapRemoteHosts(clusterConfigJSON, config);
    return config;
};

export const getStorageClasses = (config: Config): string[] => {
    const classes: Set<string> = new Set(['default']);
    const volumes = config.clusterConfig.Volumes;
    Object.keys(volumes).forEach(v => {
        Object.keys(volumes[v].StorageClasses || {}).forEach(sc => {
            if (volumes[v].StorageClasses[sc]) {
                classes.add(sc);
            }
        });
    });
    return Array.from(classes);
};

const getApiRevision = async (apiUrl: string) => {
    try {
        const dd = (await Axios.get<any>(`${apiUrl}/${DISCOVERY_DOC_PATH}`)).data;
        return parseInt(dd.revision, 10) || 0;
    } catch {
        console.warn("Unable to get API Revision number, defaulting to zero. Some features may not work properly.");
        return 0;
    }
};

const removeTrailingSlashes = (config: ClusterConfigJSON): ClusterConfigJSON => {
    const svcs: any = {};
    Object.keys(config.Services).forEach((s) => {
        svcs[s] = config.Services[s];
        if (svcs[s].hasOwnProperty('ExternalURL')) {
            svcs[s].ExternalURL = svcs[s].ExternalURL.replace(/\/+$/, '');
        }
    });
    return { ...config, Services: svcs };
};

export const fetchConfig = () => {
    return Axios
        .get<WorkbenchConfig>(WORKBENCH_CONFIG_URL + "?nocache=" + (new Date()).getTime())
        .then(response => response.data)
        .catch(() => {
            console.warn(`There was an exception getting the Workbench config file at ${WORKBENCH_CONFIG_URL}. Using defaults instead.`);
            return Promise.resolve(getDefaultConfig());
        })
        .then(workbenchConfig => {
            if (workbenchConfig.API_HOST === undefined) {
                throw new Error(`Unable to start Workbench. API_HOST is undefined in ${WORKBENCH_CONFIG_URL} or the environment.`);
            }
            return Axios.get<ClusterConfigJSON>(getClusterConfigURL(workbenchConfig.API_HOST)).then(async response => {
                const apiRevision = await getApiRevision(response.data.Services.Controller.ExternalURL.replace(/\/+$/, ''));
                const config = { ...buildConfig(response.data), apiRevision };
                const warnLocalConfig = (varName: string) => console.warn(
                    `A value for ${varName} was found in ${WORKBENCH_CONFIG_URL}. To use the Arvados centralized configuration instead, \
remove the entire ${varName} entry from ${WORKBENCH_CONFIG_URL}`);

                // Check if the workbench config has an entry for vocabulary and file viewer URLs
                // If so, use these values (even if it is an empty string), but print a console warning.
                // Otherwise, use the cluster config.
                let fileViewerConfigUrl;
                if (workbenchConfig.FILE_VIEWERS_CONFIG_URL !== undefined) {
                    warnLocalConfig("FILE_VIEWERS_CONFIG_URL");
                    fileViewerConfigUrl = workbenchConfig.FILE_VIEWERS_CONFIG_URL;
                }
                else {
                    fileViewerConfigUrl = config.clusterConfig.Workbench.FileViewersConfigURL || "/file-viewers-example.json";
                }
                config.fileViewersConfigUrl = fileViewerConfigUrl;

                if (workbenchConfig.VOCABULARY_URL !== undefined) {
                    console.warn(`A value for VOCABULARY_URL was found in ${WORKBENCH_CONFIG_URL}. It will be ignored as the cluster already provides its own endpoint, you can safely remove it.`)
                }
                config.vocabularyUrl = getVocabularyURL(workbenchConfig.API_HOST);

                return { config, apiHost: workbenchConfig.API_HOST };
            });
        });
};

// Maps remote cluster hosts and removes the default RemoteCluster entry
export const mapRemoteHosts = (clusterConfigJSON: ClusterConfigJSON, config: Config) => {
    config.remoteHosts = {};
    Object.keys(clusterConfigJSON.RemoteClusters).forEach(k => { config.remoteHosts[k] = clusterConfigJSON.RemoteClusters[k].Host; });
    delete config.remoteHosts["*"];
};

export const mockClusterConfigJSON = (config: Partial<ClusterConfigJSON>): ClusterConfigJSON => ({
    API: {
        UnfreezeProjectRequiresAdmin: false,
        MaxItemsPerResponse: 1000,
    },
    ClusterID: "",
    RemoteClusters: {},
    Services: {
        Controller: { ExternalURL: "" },
        Workbench1: { ExternalURL: "" },
        Workbench2: { ExternalURL: "" },
        Websocket: { ExternalURL: "" },
        WebDAV: { ExternalURL: "" },
        WebDAVDownload: { ExternalURL: "" },
        WebShell: { ExternalURL: "" },
    },
    Workbench: {
        DisableSharingURLsUI: false,
        ArvadosDocsite: "",
        FileViewersConfigURL: "",
        WelcomePageHTML: "",
        InactivePageHTML: "",
        SSHHelpPageHTML: "",
        SSHHelpHostSuffix: "",
        SiteName: "",
        IdleTimeout: "0s",
    },
    Login: {
        LoginCluster: "",
        Google: {
            Enable: false,
        },
        LDAP: {
            Enable: false,
        },
        OpenIDConnect: {
            Enable: false,
        },
        PAM: {
            Enable: false,
        },
        SSO: {
            Enable: false,
        },
        Test: {
            Enable: false,
        },
    },
    Collections: {
        ForwardSlashNameSubstitution: "",
        TrustAllContent: false,
    },
    Volumes: {},
    ...config
});

export const mockConfig = (config: Partial<Config>): Config => ({
    baseUrl: "",
    keepWebServiceUrl: "",
    keepWebInlineServiceUrl: "",
    remoteHosts: {},
    rootUrl: "",
    uuidPrefix: "",
    websocketUrl: "",
    workbenchUrl: "",
    workbench2Url: "",
    vocabularyUrl: "",
    fileViewersConfigUrl: "",
    loginCluster: "",
    clusterConfig: mockClusterConfigJSON({}),
    apiRevision: 0,
    ...config
});

const getDefaultConfig = (): WorkbenchConfig => {
    let apiHost = "";
    const envHost = process.env.REACT_APP_ARVADOS_API_HOST;
    if (envHost !== undefined) {
        console.warn(`Using default API host ${envHost}.`);
        apiHost = envHost;
    }
    else {
        console.warn(`No API host was found in the environment. Workbench may not be able to communicate with Arvados components.`);
    }
    return {
        API_HOST: apiHost,
        VOCABULARY_URL: undefined,
        FILE_VIEWERS_CONFIG_URL: undefined,
    };
};

export const ARVADOS_API_PATH = "arvados/v1";
export const CLUSTER_CONFIG_PATH = "arvados/v1/config";
export const VOCABULARY_PATH = "arvados/v1/vocabulary";
export const DISCOVERY_DOC_PATH = "discovery/v1/apis/arvados/v1/rest";
export const getClusterConfigURL = (apiHost: string) => `https://${apiHost}/${CLUSTER_CONFIG_PATH}?nocache=${(new Date()).getTime()}`;
export const getVocabularyURL = (apiHost: string) => `https://${apiHost}/${VOCABULARY_PATH}?nocache=${(new Date()).getTime()}`;
