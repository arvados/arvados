// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import DOMPurify from 'dompurify';

type TDomPurifyConfig = {
    ALLOWED_TAGS: string[];
    ALLOWED_ATTR: string[];
};

const domPurifyConfig: TDomPurifyConfig = {
    ALLOWED_TAGS: [
        'a',
        'b',
        'blockquote',
        'br',
        'code',
        'del',
        'dd',
        'dl',
        'dt',
        'em',
        'h1',
        'h2',
        'h3',
        'h4',
        'h5',
        'h6',
        'hr',
        'i',
        'img',
        'kbd',
        'li',
        'ol',
        'p',
        'pre',
        's',
        'del',
        'section',
        'span',
        'strong',
        'sub',
        'sup',
        'ul',
    ],
    ALLOWED_ATTR: ['src', 'width', 'height', 'href', 'alt', 'title', 'style' ],
};

export const sanitizeHTML = (dirtyString: string): string => DOMPurify.sanitize(dirtyString, domPurifyConfig);

// Config for further sanitization of table cell content; used for display only.
const tableCellPurifyConfig: TDomPurifyConfig = {
    ALLOWED_TAGS: [
        'a',
        'b',
        'br',
        'code',
        'del',
        'em',
        'i',
        'kbd',
        'pre',
        's',
        'span',
        'strong',
        'sub',
        'sup',
    ],
    ALLOWED_ATTR: ['href', 'title'],
};

export const sanitizeTableCellContent = (dirtyString: string): string => DOMPurify.sanitize(dirtyString, tableCellPurifyConfig);
