// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// tests if string matches xxxxx-xxxxx-xxxxxxxxxxxxxxx where x = any alphanumeric char
export const isUUID = (str: string) => {
    const uuidRegex = /^[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}$/i;
    return uuidRegex.test(str);
  }