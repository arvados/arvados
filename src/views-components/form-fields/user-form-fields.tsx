// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field } from "redux-form";
import { TextField } from "components/text-field/text-field";
import { USER_EMAIL_VALIDATION, CHOOSE_VM_VALIDATION } from "validators/validators";
import { NativeSelectField } from "components/select-field/select-field";
import { InputLabel } from "@material-ui/core";
import { VirtualMachinesResource } from "models/virtual-machines";
import { VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD } from "store/virtual-machines/virtual-machines-actions";
import { GroupArrayInput } from "views-components/virtual-machines-dialog/group-array-input";

interface VirtualMachinesProps {
    data: {
        items: VirtualMachinesResource[];
    };
}

export const UserEmailField = () =>
    <Field
        name='email'
        component={TextField as any}
        validate={USER_EMAIL_VALIDATION}
        autoFocus={true}
        label="Email" />;

export const RequiredUserVirtualMachineField = ({ data }: VirtualMachinesProps) =>
    <div style={{ marginBottom: '21px' }}>
        <InputLabel>Virtual Machine</InputLabel>
        <Field
            name={VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD}
            component={NativeSelectField as any}
            validate={CHOOSE_VM_VALIDATION}
            items={getVirtualMachinesList(data.items)} />
    </div>;

export const UserVirtualMachineField = ({ data }: VirtualMachinesProps) =>
    <div style={{ marginBottom: '21px' }}>
        <InputLabel>Virtual Machine</InputLabel>
        <Field
            name={VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD}
            component={NativeSelectField as any}
            items={getVirtualMachinesList(data.items)} />
    </div>;

export const UserGroupsVirtualMachineField = () =>
    <GroupArrayInput
        name={VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD}
        input={{id:"Add groups to VM login (eg: docker, sudo)", disabled:false}}
        required={false}
    />

const getVirtualMachinesList = (virtualMachines: VirtualMachinesResource[]) =>
    [{ key: "", value: "" }].concat(virtualMachines.map(it => ({ key: it.uuid, value: it.hostname })));
