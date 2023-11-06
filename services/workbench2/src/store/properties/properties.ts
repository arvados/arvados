// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type PropertiesState = { [key: string]: any };

export const getProperty = <T>(id: string) =>
    (state: PropertiesState): T | undefined =>
        state[id];

export const setProperty = <T>(id: string, data: T) =>
    (state: PropertiesState) => ({
        ...state,
        [id]: data
    });

export const deleteProperty = (id: string) =>
    (state: PropertiesState) => {
        const newState = { ...state };
        delete newState[id];
        return newState;
    };

