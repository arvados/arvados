// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const getBuildInfo = (): string => {
    if (process.env.REACT_APP_VERSION) {
      return "v" + process.env.REACT_APP_VERSION;
    } else {
      const getBuildNumber = "BN-" + (process.env.REACT_APP_BUILD_NUMBER || "dev");
      const getGitCommit = "GIT-" + (process.env.REACT_APP_GIT_COMMIT || "latest").substring(0, 7);
      return getBuildNumber + " / " + getGitCommit;
    }
};

export const PROPERTY_CONTAINS_VALUE_MIN_API_REVISION = 20200212;

export const GROUP_CONTENTS_INCLUDE_CONTAINER_UUID_MIN_API_REVISION = 20240627;
