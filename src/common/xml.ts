// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const getTagValue = (document: Document | Element, tagName: string, defaultValue: string) => {
    const [el] = Array.from(document.getElementsByTagName(tagName));
    return decodeURI(el ? el.innerHTML : defaultValue);
};
