// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "axios";

export const WORKBENCH_CONFIG_URL = process.env.REACT_APP_ARVADOS_CONFIG_URL || "/config.json";

interface WorkbenchConfig {
    API_HOST: string;
    VOCABULARY_URL: string;
    FILE_VIEWERS_CONFIG_URL: string;
}

export interface ClusterConfigJSON {
    ClusterID: string;
    RemoteClusters:  {
        [key: string]: {
            ActivateUsers: boolean
            Host: string
            Insecure: boolean
            Proxy: boolean
            Scheme: string
        }
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
        }
    };
    Workbench: {
        ArvadosDocsite: string;
        VocabularyURL: string;
        FileViewersConfigURL: string;
    };
}

export class Config {
    baseUrl: string;
    keepWebServiceUrl: string;
    remoteHosts: {
        [key: string]: string
    };
    rootUrl: string;
    uuidPrefix: string;
    websocketUrl: string;
    workbenchUrl: string;
    workbench2Url: string;
    vocabularyUrl: string;
    fileViewersConfigUrl: string;
}

export const fetchConfig = () => {
    return Axios
        .get<WorkbenchConfig>(WORKBENCH_CONFIG_URL + "?nocache=" + (new Date()).getTime())
        .then(response => response.data)
        .catch(() => Promise.resolve(getDefaultConfig()))
        .then(workbenchConfig => Axios
            .get<ClusterConfigJSON>(getClusterConfigURL(workbenchConfig.API_HOST))
            .then(response => {

                const config = new Config();
                const clusterConfigJSON = response.data;
                const docsite = clusterConfigJSON.Workbench.ArvadosDocsite;
                const warnDeprecation = (varName: string) => console.warn(
`A value for ${varName} was found in ${WORKBENCH_CONFIG_URL}. This configuration is deprecated. \
Please use the centralized configuration instead. See more at ${docsite}admin/config-migration.html`);

                // Check if the workbench config has an entry for vocabulary and file viewer URLs
                // If so, use these values (even if it is an empty string), but print a console warning.
                // Otherwise, use the cluster config or default values.
                let fileViewerConfigUrl;
                if (workbenchConfig.FILE_VIEWERS_CONFIG_URL !== undefined) {
                    warnDeprecation("FILE_VIEWERS_CONFIG_URL");
                    fileViewerConfigUrl = workbenchConfig.FILE_VIEWERS_CONFIG_URL;
                }
                else {
                    fileViewerConfigUrl = clusterConfigJSON.Workbench.FileViewersConfigURL || "/file-viewers-example.json";
                }
                config.fileViewersConfigUrl = fileViewerConfigUrl;

                let vocabularyUrl;
                if (workbenchConfig.VOCABULARY_URL !== undefined) {
                    warnDeprecation("VOCABULARY_URL");
                    vocabularyUrl = workbenchConfig.VOCABULARY_URL;
                }
                else {
                    vocabularyUrl = clusterConfigJSON.Workbench.VocabularyURL || "/vocabulary-example.json";
                }
                config.vocabularyUrl = vocabularyUrl;

                config.rootUrl = clusterConfigJSON.Services.Controller.ExternalURL;
                config.baseUrl = `${config.rootUrl}/${ARVADOS_API_PATH}`;
                config.uuidPrefix = clusterConfigJSON.ClusterID;
                config.websocketUrl = clusterConfigJSON.Services.Websocket.ExternalURL;
                config.workbench2Url = clusterConfigJSON.Services.Workbench2.ExternalURL;
                config.workbenchUrl = clusterConfigJSON.Services.Workbench1.ExternalURL;
                config.keepWebServiceUrl =  clusterConfigJSON.Services.WebDAV.ExternalURL;
                mapRemoteHosts(clusterConfigJSON, config);

                return { config, apiHost: workbenchConfig.API_HOST };
            })
        );
};

// Maps remote cluster hosts and removes the default RemoteCluster entry
export const mapRemoteHosts = (clusterConfigJSON: ClusterConfigJSON, config: Config) => {
    config.remoteHosts = {};
    Object.keys(clusterConfigJSON.RemoteClusters).forEach (k => { config.remoteHosts[k] = clusterConfigJSON.RemoteClusters[k].Host; });
    delete config.remoteHosts["*"];
};

export const mockConfig = (config: Partial<Config>): Config => ({
    baseUrl: "",
    keepWebServiceUrl: "",
    remoteHosts: {},
    rootUrl: "",
    uuidPrefix: "",
    websocketUrl: "",
    workbenchUrl: "",
    workbench2Url: "",
    vocabularyUrl: "",
    fileViewersConfigUrl: ""
});

const getDefaultConfig = (): WorkbenchConfig => ({
    API_HOST: process.env.REACT_APP_ARVADOS_API_HOST || "",
    VOCABULARY_URL: "",
    FILE_VIEWERS_CONFIG_URL: "",
});

export const ARVADOS_API_PATH = "arvados/v1";
export const CLUSTER_CONFIG_URL = "arvados/v1/config";
export const getClusterConfigURL = (apiHost: string) => `${window.location.protocol}//${apiHost}/${CLUSTER_CONFIG_URL}?nocache=${(new Date()).getTime()}`;