// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import IconBase, { IconTypes } from '../../icon/icon';
import Attribute from '../../attribute/attribute';
import AbstractItem from './abstract-item';
import { ProjectResource } from '../../../models/project';

export default class ProjectItem extends AbstractItem<ProjectResource> {

    getIcon(): IconTypes {
        return IconTypes.PROJECT;
    }

    buildDetails(): React.ReactElement<any> {
        return <div>
            <Attribute label='Type' value='Project' />
            <Attribute label='Size' value='---' />
            <Attribute label="Location">
                <IconBase icon={IconTypes.FOLDER} />
                Projects
                </Attribute>
            <Attribute label='Owner' value='me' />
            <Attribute label='Last modified' value='5:25 PM 5/23/2018' />
            <Attribute label='Created at' value='1:25 PM 5/23/2018' />
            <Attribute label='File size' value='1.4 GB' />
        </div>;
    }
}