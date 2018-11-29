// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { ServiceRepository } from '~/services/services';
import { propertiesActions } from '~/store/properties/properties-actions';
import { VOCABULARY_PROPERTY_NAME, DEFAULT_VOCABULARY } from './vocabulary-selctors';
import { isVocabulary } from '~/models/vocabulary';

export const loadVocabulary = async (dispatch: Dispatch, _: {}, { vocabularyService }: ServiceRepository) => {
    const vocabulary = await vocabularyService.getVocabulary();

    dispatch(propertiesActions.SET_PROPERTY({
        key: VOCABULARY_PROPERTY_NAME,
        value: isVocabulary(vocabulary)
            ? vocabulary
            : DEFAULT_VOCABULARY,
    }));
};
