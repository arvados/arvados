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
        expect(state[id]).to.deep.equal({ open: true, data });
    });

    it('CLOSE_DIALOG', () => {
        const id = 'test id';
        const state = dialogReducer({}, dialogActions.CLOSE_DIALOG({ id }));
        expect(state[id]).to.deep.equal({ open: false, data: {} });
    });
    
    it('CLOSE_DIALOG persist data', () => {
        const id = 'test id';
        const [newState] = [{}]
            .map(state => dialogReducer(state, dialogActions.OPEN_DIALOG({ id, data: 'test data' })))
            .map(state => dialogReducer(state, dialogActions.CLOSE_DIALOG({ id })));
        
        expect(newState[id]).to.deep.equal({ open: false, data: 'test data' });
    });
});
