// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { defineConfig } from "cypress";

export default defineConfig({
  chromeWebSecurity: false,
  viewportWidth: 1920,
  viewportHeight: 1080,
  downloadsFolder: "cypress/downloads",
  videoCompression: false,

  // projectId is for use with Cypress Cloud/CI, which we are currently not using
  // projectId: "pzrqer",

  e2e: {
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
    setupNodeEvents(on, config) {
      return require("./cypress/plugins/index.js")(on, config);
    },
    baseUrl: "https://localhost:3000/",
    experimentalRunAllSpecs: true,
    // The 2 options below make Electron crash a lot less and Firefox behave better
    experimentalMemoryManagement: true,
    numTestsKeptInMemory: 0,
  },

  component: {
    devServer: {
      framework: "react",
      bundler: "webpack",
      webpackConfig: require("./config/webpack.config"),
    },
  },
});
