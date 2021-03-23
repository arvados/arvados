// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const getTagValue = (document: Document | Element, tagName: string, defaultValue: string) => {
    const [el] = Array.from(document.getElementsByTagName(tagName));
    const URI = el ? htmlDecode(el.innerHTML) : defaultValue;

    try {
        return decodeURI(URI);
    } catch(e) {}

    return URI;
};

const htmlDecode = (input: string) => {
    const out = input.split(' ').map((i) => {
        const doc = new DOMParser().parseFromString(i, "text/html");
        if (doc.documentElement !== null) {
            return doc.documentElement.textContent || '';
        }
        return '';
    });
    return out.join(' ');
};
