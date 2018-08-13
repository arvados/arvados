// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "../../node_modules/axios";

export const CONFIG_URL = process.env.REACT_APP_ARVADOS_CONFIG_URL || "/config.json";

export interface Config {
    apiHost: string;
    keepWebHost: string;
}

export const fetchConfig = () => {
    return Axios
        .get<Config>(CONFIG_URL + "?nocache=" + (new Date()).getTime())
        .then(response => response.data)
        .catch(() => Promise.resolve(getDefaultConfig()))
        .then(mapConfig);
};

const mapConfig = (config: Config): Config => ({
    ...config,
    apiHost: addProtocol(config.apiHost),
    keepWebHost: addProtocol(config.keepWebHost)
});

const getDefaultConfig = (): Config => ({
    apiHost: process.env.REACT_APP_ARVADOS_API_HOST || "",
    keepWebHost: process.env.REACT_APP_ARVADOS_KEEP_WEB_HOST || ""
});

const addProtocol = (url: string) => `${window.location.protocol}//${url}`;
