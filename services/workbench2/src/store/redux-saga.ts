// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { call, all, spawn } from "redux-saga/effects";
import {
    setTreePickerProjectSearchWatcher,
    loadProjectWatcher,
    loadSearchWatcher,
    refreshTreePickerWatcher,
    setTreePickerCollectionFilterWatcher,
    loadFavoritesProjectWatcher,
} from "./tree-picker/tree-picker-actions";

/**
* Auto restart sagas with error logging
*/
export const rootSaga = function* () {
   const sagas = [
       setTreePickerProjectSearchWatcher,
       setTreePickerCollectionFilterWatcher,
       refreshTreePickerWatcher,
       loadProjectWatcher,
       loadSearchWatcher,
       loadFavoritesProjectWatcher,
   ];

   yield all(sagas.map(saga =>
       spawn(function* () {
           while (true) {
               try {
                   yield call(saga);
                   break;
               } catch (e) {
                   console.error(e);
               }
           }
       }))
   );
}
