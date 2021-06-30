// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { Field } from "redux-form";
import { ResourcesState, getResource } from "store/resources/resources";
import { GroupResource } from "models/group";
import { TextField } from "components/text-field/text-field";
import { getUserUuid } from "common/getuser";

interface ResourceParentFieldProps {
    resources: ResourcesState;
    userUuid: string|undefined;
}

export const ResourceParentField = connect(
    (state: RootState) => {
        return {
            resources: state.resources,
            userUuid: getUserUuid(state),
        };
    })
    ((props: ResourceParentFieldProps) =>
        <span data-cy='parent-field'><Field
            name='ownerUuid'
            disabled={true}
            label='Parent project'
            format={
                (value, name) => {
                    if (value === props.userUuid) {
                        return 'Home project';
                    }
                    const rsc = getResource<GroupResource>(value)(props.resources);
                    if (rsc !== undefined) {
                        return `${rsc.name} (${rsc.uuid})`;
                    }
                    return value;
                }
            }
            component={TextField} /></span>
    );
