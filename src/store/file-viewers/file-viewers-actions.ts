// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { ServiceRepository } from '~/services/services';
import { propertiesActions } from '~/store/properties/properties-actions';
import { FILE_VIEWERS_PROPERTY_NAME, DEFAULT_FILE_VIEWERS } from '~/store/file-viewers/file-viewers-selectors';
import { FileViewerList } from '~/models/file-viewers-config';

export const loadFileViewersConfig = async (dispatch: Dispatch, _: {}, { fileViewersConfig }: ServiceRepository) => {
    
    let config: FileViewerList;
    try{
        config = await fileViewersConfig.get();
    } catch (e){
        config = DEFAULT_FILE_VIEWERS;
    }

    dispatch(propertiesActions.SET_PROPERTY({
        key: FILE_VIEWERS_PROPERTY_NAME,
        value: config,
    }));

};
