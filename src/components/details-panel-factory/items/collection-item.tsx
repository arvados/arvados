// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import IconBase, { IconTypes } from '../../icon/icon';
import Attribute from '../../attribute/attribute';
import AbstractItem from './abstract-item';
import { CollectionResource } from '../../../models/collection';

export default class CollectionItem extends AbstractItem<CollectionResource> {

    getIcon(): IconTypes {
        return IconTypes.COLLECTION;
    }

    buildDetails(): React.ReactElement<any> {
        return <div>
           <Attribute label='Type' value='Data Collection' />
            <Attribute label='Size' value='---' />
            <Attribute label="Location">
                <IconBase icon={IconTypes.FOLDER} />
                Collection
            </Attribute>
            <Attribute label='Owner' value='me' />
            <Attribute label='Last modified' value='5:25 PM 5/23/2018' />
            <Attribute label='Created at' value='1:25 PM 5/23/2018' />
            <Attribute label='Number of files' value='20' />
            <Attribute label='Content size' value='54 MB' />
            <Attribute label='Collection UUID' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
            <Attribute label='Content address' link='http://www.google.pl' value='nfnz05wp63ibf8w' />
            <Attribute label='Creator' value='Chrystian' />
            <Attribute label='Used by' value='---' />
        </div>;
    }
}