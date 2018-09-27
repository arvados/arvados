// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { ServiceRepository } from "~/services/services";
import { RootState } from '~/store/store';

export const runProcessPanelActions = unionize({
    CHANGE_STEP: ofType<number>()
});

export type RunProcessPanelAction = UnionOf<typeof runProcessPanelActions>;

export const loadRunProcessPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        
    };

export const goToStep = (step: number) => runProcessPanelActions.CHANGE_STEP(step);