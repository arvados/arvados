// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const mockServices = {
    collectionService: {
        files: ()=>({ then: (callback) => callback([{ name: 'banner.html' }]) }),
        getFileContents: ()=>({ then: (callback) => callback('<h1>Test banner message</h1>') }),
    },
};

const serviceProvider = {
    getServices: () => {
        return mockServices
    },
}

export default serviceProvider;