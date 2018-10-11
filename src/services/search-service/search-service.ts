// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export class SearchQueriesService {
    private recentQueries: string[] = this.getRecentQueries();

    saveRecentQuery(query: string) {
        if (this.recentQueries.length >= 5) {
            this.recentQueries.shift();
            this.recentQueries.push(query);
        } else {
            this.recentQueries.push(query);
        }
        localStorage.setItem('recentQueries', JSON.stringify(this.recentQueries));
    }

    getRecentQueries() {
        return JSON.parse(localStorage.getItem('recentQueries') || '[]') as string[];
    }
}