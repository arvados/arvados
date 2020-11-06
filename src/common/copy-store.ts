// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const STORE_COPY_KEY = 'storeCopy';

export const copyStore = (store: any) => {
    const { localStorage } = window;
    const state = store.getState();
    const storeCopy = JSON.parse(JSON.stringify(state));
    storeCopy.router.location.pathname = '/';

    if (localStorage) {
        localStorage.setItem(STORE_COPY_KEY, JSON.stringify(storeCopy));
    }
};

export const restoreStore = () => {
    let storeCopy = null;
    const { localStorage } = window;

    if (localStorage && localStorage.getItem(STORE_COPY_KEY)) {
        storeCopy = localStorage.getItem(STORE_COPY_KEY);
        localStorage.removeItem(STORE_COPY_KEY);
    }

    return storeCopy;
};
