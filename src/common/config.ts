// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "../../node_modules/axios";

export const CONFIG_URL = process.env.REACT_APP_ARVADOS_CONFIG_URL || "/config.json";

export interface Config {
    API_HOST: string;
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
    API_HOST: addProtocol(config.API_HOST)
});

const getDefaultConfig = (): Config => ({
    API_HOST: process.env.REACT_APP_ARVADOS_API_HOST || ""
});

const addProtocol = (url: string) => `${window.location.protocol}//${url}`;
