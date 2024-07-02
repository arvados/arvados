// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { defineConfig } from 'cypress'

export default defineConfig({
  chromeWebSecurity: false,
  viewportWidth: 1920,
  viewportHeight: 1080,
  downloadsFolder: 'cypress/downloads',
  videoCompression: false,
  e2e: {
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
      setupNodeEvents(on, config) {
          require("cypress-fail-fast/plugin")(on, config);
          require('./cypress/plugins/index.js')(on, config)
          return config;
      },
      baseUrl: 'https://localhost:3000/',
      experimentalRunAllSpecs: true,
      // The 2 options below make Electron crash a lot less and Firefox behave better
      experimentalMemoryManagement: true,
      numTestsKeptInMemory: 2,
  },
})
