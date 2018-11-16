// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { virtualMachinesActions, VirtualMachineActions } from '~/store/virtual-machines/virtual-machines-actions';
import { ListResults } from '~/services/common-service/common-resource-service';
import { VirtualMachinesLoginsResource } from '~/models/virtual-machines';

interface VirtualMachines {
    date: string;
    virtualMachines: ListResults<any>;
    logins: VirtualMachinesLoginsResource[];
}

const initialState: VirtualMachines = {
    date: '',
    virtualMachines: {
        kind: '',
        offset: 0,
        limit: 0,
        itemsAvailable: 0,
        items: []
    },
    logins: []
};

export const virtualMachinesReducer = (state = initialState, action: VirtualMachineActions): VirtualMachines =>
    virtualMachinesActions.match(action, {
        SET_REQUESTED_DATE: date => ({ ...state, date }),
        SET_VIRTUAL_MACHINES: virtualMachines => ({ ...state, virtualMachines }),
        SET_LOGINS: logins => ({ ...state, logins }),
        default: () => state
    });
