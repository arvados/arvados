// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

class ServicesProvider {

    private static instance: ServicesProvider;

    private services;

    private constructor() {}

    public static getInstance(): ServicesProvider {
        if (!ServicesProvider.instance) {
            ServicesProvider.instance = new ServicesProvider();
        }

        return ServicesProvider.instance;
    }

    public setServices(newServices): void {
        if (!this.services) {
            this.services = newServices;
        }
    }

    public getServices() {
        if (!this.services) {
            throw "Please check if services have been set in the index.ts before the app is initiated"; // eslint-disable-line no-throw-literal
        }
        return this.services;
    }
}

export default ServicesProvider.getInstance();
