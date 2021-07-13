// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { getResource } from 'store/resources/resources';
import { Resource } from 'models/resource';

interface WithResourceProps {
    resource?: Resource;
}

export const withResource = (component: React.ComponentType<WithResourceProps & { uuid: string }>) =>
    connect<WithResourceProps>(
        (state: RootState, props: { uuid: string }): WithResourceProps => ({
            resource: getResource(props.uuid)(state.resources)
        })
    )(component);

export const getDataFromResource = (property: string, resource?: Resource) => {
    return resource && resource[property] ? resource[property] : '(none)';
};

export const withResourceData = (property: string, render: (data: any) => React.ReactElement<any>) =>
    withResource(({ resource }) => render(getDataFromResource(property, resource)));
