// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from 'axios';
import { Vocabulary } from 'models/vocabulary';

export class VocabularyService {
    constructor(
        private url: string
    ) { }

    async getVocabulary() {
        const response = await Axios
            .get<Vocabulary>(this.url);
        return response.data;
    }
}
