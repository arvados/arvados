// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

/// <reference types="cypress" />
// ***********************************************************
// This example plugins/index.js can be used to load plugins
//
// You can change the location of this file or turn off loading
// the plugins file with the 'pluginsFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/plugins-guide
// ***********************************************************

// This function is called when a project is opened or re-opened (e.g. due to
// the project's config changing)

const fs = require('fs');
const path = require('path');

/**
 * @type {Cypress.PluginConfig}
 */
module.exports = (on, config) => {
  // `on` is used to hook into various events Cypress emits
  // `config` is the resolved Cypress config
  on("before:browser:launch", (browser = {}, launchOptions) => {
    const downloadDirectory = path.join(__dirname, "..", "downloads");
    if (browser.family === 'chromium' && browser.name !== 'electron') {
     launchOptions.preferences.default["download"] = {
      default_directory: downloadDirectory
     };
    }
    return launchOptions;
  });

  on('task', {
    clearDownload({ filename }) {
      fs.unlinkSync(filename);
      return null;
    }
  });
}
