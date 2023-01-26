// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionDirectory, CollectionFile } from "models/collection-file";
import { Middleware, Store } from "redux";
import { ServiceRepository } from "services/services";
import { RootState } from "store/store";
import tippy, { createSingleton } from 'tippy.js';
import 'tippy.js/dist/tippy.css';

let running = false;
let tooltipsContents = null;
let tooltipsFetchFailed = false;
const TOOLTIP_LOCAL_STORAGE_KEY = "TOOLTIP_LOCAL_STORAGE_KEY";

export const tooltipsMiddleware = (services: ServiceRepository): Middleware => (store: Store) => next => action => {
    const state: RootState = store.getState();
    const result = localStorage.getItem(TOOLTIP_LOCAL_STORAGE_KEY);
    const { BannerURL } = (state.auth.config.clusterConfig.Workbench as any);

    let bannerUUID = !!BannerURL ? BannerURL : 'tordo-4zz18-1buneu6sb8zxiti';

    if (bannerUUID && !tooltipsContents && !result && !tooltipsFetchFailed && !running) {
        running = true;
        fetchTooltips(services, bannerUUID);
    } else if (tooltipsContents && !result && !tooltipsFetchFailed) {
        applyTooltips();
    }

    return next(action);
};

const fetchTooltips = (services, bannerUUID) => {
    services.collectionService.files(bannerUUID)
        .then(results => {
            const tooltipsFile: CollectionDirectory | CollectionFile | undefined = results.find(({ name }) => name === 'tooltips.json');

            if (tooltipsFile) {
                running = true;
                services.collectionService.getFileContents(tooltipsFile as CollectionFile)
                    .then(data => {
                        tooltipsContents = JSON.parse(data);
                        applyTooltips();
                    })
                    .catch(() => {})
                    .finally(() => {
                        running = false;
                    });
            }  else {
                tooltipsFetchFailed = true;
            }
        })
        .catch(() => {})
        .finally(() => {
            running = false;
        });
};

const applyTooltips = () => {
    const tippyInstances: any[] = Object.keys(tooltipsContents as any)
        .map((key) => {
            const content = (tooltipsContents as any)[key]
            const element = document.querySelector(key);

            if (element) {
                const hasTippyAttatched = !!(element as any)._tippy;

                if (!hasTippyAttatched && tooltipsContents) {
                    return tippy(element as any, { content });
                }
            }

            return null;
        })
        .filter(data => !!data);

        createSingleton(tippyInstances, {delay: 10});
};