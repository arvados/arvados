// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "../../node_modules/axios";

export const CONFIG_URL = process.env.REACT_APP_ARVADOS_CONFIG_URL || "/config.json";

export interface Config {
    API_HOST: string;
}

const defaultConfig: Config = {
    API_HOST: process.env.REACT_APP_ARVADOS_API_HOST || ""
};

export const fetchConfig = () => {
    return Axios
        .get<Config>(CONFIG_URL + "?nocache=" + (new Date()).getTime())
        .then(response => response.data)
        .catch(() => Promise.resolve(defaultConfig));
};

