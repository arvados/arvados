// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface ParsedSearchQuery {
    tokens: string[];
    searchString: string;
}

export const findToken = (query: string, patterns: RegExp[]) => {
    for (const pattern of patterns) {
        const match = query.match(pattern);
        if (match) {
            return match[0];
        }
    }
    return null;
};

export const findAllTokens = (query: string, patterns: RegExp[]): string[] => {
    const token = findToken(query, patterns);
    return token
        ? [token].concat(findAllTokens(query.replace(token, ''), patterns))
        : [];
};

export const findSearchString = (query: string, tokens: string[]) => {
    return tokens.reduce((q, token) => q.replace(token, ''), query);
};

export const parseSearchQuery = (patterns: RegExp[]) => (query: string): ParsedSearchQuery => {
    const tokens = findAllTokens(query, patterns);
    const searchString = findSearchString(query, tokens);
    return { tokens, searchString };
};
