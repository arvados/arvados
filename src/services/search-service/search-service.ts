// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { SearchBarAdvancedFormData } from '~/models/search-bar';

export class SearchService {
    private recentQueries = this.getRecentQueries();
    private savedQueries: SearchBarAdvancedFormData[] = this.getSavedQueries();

    saveRecentQuery(query: string) {
        if (this.recentQueries.length >= MAX_NUMBER_OF_RECENT_QUERIES) {
            this.recentQueries.shift();
        }
        this.recentQueries.push(query);
        localStorage.setItem('recentQueries', JSON.stringify(this.recentQueries));
    }

    getRecentQueries(): string[] {
        return JSON.parse(localStorage.getItem('recentQueries') || '[]');
    }

    saveQuery(data: SearchBarAdvancedFormData) {
        this.savedQueries.push({...data});
        localStorage.setItem('savedQueries', JSON.stringify(this.savedQueries));
    }

    editSavedQueries(data: SearchBarAdvancedFormData) {
        const itemIndex = this.savedQueries.findIndex(item => item.queryName === data.queryName);
        this.savedQueries[itemIndex] = {...data};
        localStorage.setItem('savedQueries', JSON.stringify(this.savedQueries));
    }

    getSavedQueries() {
        return JSON.parse(localStorage.getItem('savedQueries') || '[]') as SearchBarAdvancedFormData[];
    }

    deleteSavedQuery(id: number) {
        this.savedQueries.splice(id, 1);
        localStorage.setItem('savedQueries', JSON.stringify(this.savedQueries));
    }
}

const MAX_NUMBER_OF_RECENT_QUERIES = 5;
