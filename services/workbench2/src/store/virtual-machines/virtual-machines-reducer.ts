// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { virtualMachinesActions, VirtualMachineActions } from 'store/virtual-machines/virtual-machines-actions';
import { ListResults } from 'services/common-service/common-service';
import { VirtualMachineLogins } from 'models/virtual-machines';

interface VirtualMachines {
    date: string;
    virtualMachines: ListResults<any>;
    logins: VirtualMachineLogins;
    links: ListResults<any>;
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
    logins: {
        kind: '',
        items: []
    },
    links: {
        kind: '',
        offset: 0,
        limit: 0,
        itemsAvailable: 0,
        items: []
    }
};

export const virtualMachinesReducer = (state = initialState, action: VirtualMachineActions): VirtualMachines =>
    virtualMachinesActions.match(action, {
        SET_REQUESTED_DATE: date => ({ ...state, date }),
        SET_VIRTUAL_MACHINES: virtualMachines => ({ ...state, virtualMachines }),
        SET_LOGINS: logins => ({ ...state, logins }),
        SET_LINKS: links => ({ ...state, links }),
        default: () => state
    });
