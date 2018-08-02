// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogReducer } from "./dialog-reducer";
import { dialogActions } from "./dialog-actions";

describe('DialogReducer', () => {
    it('OPEN_DIALOG', () => {
        const id = 'test id';
        const data = 'test data';
        const state = dialogReducer({}, dialogActions.OPEN_DIALOG({ id, data }));
        expect(state[id]).toEqual({ open: true, data });
    });

    it('CLOSE_DIALOG', () => {
        const id = 'test id';
        const state = dialogReducer({}, dialogActions.CLOSE_DIALOG({ id }));
        expect(state[id]).toEqual({ open: false });
    });
});
