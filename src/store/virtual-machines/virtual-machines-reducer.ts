// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { virtualMachinesAction, VirtualMachineActions } from '~/store/virtual-machines/virtual-machines-actions';

interface VirtualMachines {
    date: string;
}

const initialState: VirtualMachines = {
    date: ''
};

export const virtualMachinesReducer = (state = initialState, action: VirtualMachineActions): VirtualMachines =>
    virtualMachinesAction.match(action, {
        SET_REQUESTED_DATE: date => ({ ...state, date }),
        default: () => state
    });
