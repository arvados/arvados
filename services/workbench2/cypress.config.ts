// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { defineConfig } from "cypress";
import baseWebpackConfig from "./config/webpack.config";
import path from "path";
import EventEmitter from "events";

// Increase the default max listeners to avoid warnings
// this doesn't matter when running the entire suite,
// but necessary when running a single test repeatedly
EventEmitter.defaultMaxListeners = 100;

const webpackConfig = baseWebpackConfig("development");

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
      webpackConfig: {
        ...webpackConfig,
        resolve: {
          ...webpackConfig.resolve,
          alias: {
            ...webpackConfig.resolve.alias,
            // redirect imported modules to the mock files for the cypress tests
            "common/service-provider": path.resolve("src/cypress/mocks/service-provider.ts"),
          },
        },
      },
    },
  },
});
