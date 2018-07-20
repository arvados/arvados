// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const isUniqName = (error: string) => {
    return sleep(1000).then(() => {
      if (error.includes("UniqueViolation")) {
        throw { error: 'Project with this name already exists.' };
      }
    });
  };

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
