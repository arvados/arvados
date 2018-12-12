// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { ServiceRepository } from '~/services/services';
import { propertiesActions } from '~/store/properties/properties-actions';
import { FILE_VIEWERS_PROPERTY_NAME } from '~/store/file-viewers/file-viewers-selectors';

export const loadFileViewersConfig = async (dispatch: Dispatch, _: {}, { fileViewersConfig }: ServiceRepository) => {

    const config = await fileViewersConfig.get();

    dispatch(propertiesActions.SET_PROPERTY({
        key: FILE_VIEWERS_PROPERTY_NAME,
        value: config,
    }));

};
