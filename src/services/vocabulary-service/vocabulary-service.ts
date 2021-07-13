// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from 'axios';
import { Vocabulary } from 'models/vocabulary';

export class VocabularyService {
    constructor(
        private url: string
    ) { }

    getVocabulary() {
        return Axios
            .get<Vocabulary>(this.url)
            .then(response => response.data);
    }
}
