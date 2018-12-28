// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { USER_EMAIL_VALIDATION, USER_LENGTH_VALIDATION } from "~/validators/validators";
import { NativeSelectField } from "~/components/select-field/select-field";
import { InputLabel } from "@material-ui/core";
import { VirtualMachinesResource } from "~/models/virtual-machines";

export const UserEmailField = () =>
    <Field
        name='email'
        component={TextField}
        validate={USER_EMAIL_VALIDATION}
        autoFocus={true}
        label="Email" />;

export const UserVirtualMachineField = ({ data }: any) =>
    <div style={{ marginBottom: '21px' }}>
        <InputLabel>Virtual Machine</InputLabel>
        <Field
            name='virtualMachine'
            component={NativeSelectField}
            validate={USER_LENGTH_VALIDATION}
            items={getVirtualMachinesList(data.items)} />
    </div>;

export const UserGroupsVirtualMachineField = () =>
    <Field
        name='groups'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        label="Groups for virtual machine (comma separated list)" />;

const getVirtualMachinesList = (virtualMachines: VirtualMachinesResource[]) => {
    const mappedVirtualMachines = virtualMachines.map(it => ({ key: it.hostname, value: it.hostname }));
    return mappedVirtualMachines;
};
