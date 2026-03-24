// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ResourcesState, getResource } from "store/resources/resources";
import { GroupResource } from "models/group";
import { getUserUuid } from "common/getuser";
import { DialogTextField } from "components/dialog-form/dialog-text-field";

interface ResourceParentFieldProps {
    resources: ResourcesState;
    userUuid: string|undefined;
    ownerUuid: string;
}

export const ResourceParentField = connect(
    (state: RootState) => {
        return {
            resources: state.resources,
            userUuid: getUserUuid(state),
        };
    })
    ((props: ResourceParentFieldProps) => {
        const format = (value: string) => {
            if (value === props.userUuid) {
                return 'Home project';
            }
            const rsc = getResource<GroupResource>(value)(props.resources);
            if (rsc !== undefined) {
                return `${rsc.name} (${rsc.uuid})`;
            }
            return value;
        }

        return <span data-cy='parent-field'>
            <DialogTextField
                label='Parent project'
                validators={[]}
                defaultValue={format(props.ownerUuid || '')}
                setValue={() => { /* no-op */ }}
                disabled={true}
            />
        </span>
    }
);
