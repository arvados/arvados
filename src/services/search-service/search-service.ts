// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export class SearchQueriesService {
    private recentQueries: string[] = this.getRecentQueries();
    private savedQueries: string[] = this.getSavedQueries();

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

    saveQuery(query: string) {
        this.savedQueries.push(query);
        localStorage.setItem('savedQueries', JSON.stringify(this.savedQueries));
    }

    getSavedQueries() {
        return JSON.parse(localStorage.getItem('savedQueries') || '[]') as string[];
    }

    deleteSavedQuery(id: number) {
        const queryToDelete = this.savedQueries[id];
        const restQueries = this.savedQueries.filter(query => query !== queryToDelete);
        return localStorage.setItem('savedQueries', JSON.stringify(restQueries));
    }
}