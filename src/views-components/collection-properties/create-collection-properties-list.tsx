// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import {
    withStyles,
    StyleRulesCallback,
    WithStyles,
} from '@material-ui/core';
import { RootState } from 'store/store';
import {
    removePropertyFromCreateCollectionForm,
    COLLECTION_CREATE_FORM_SELECTOR,
    CollectionProperties
} from 'store/collections/collection-create-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { getPropertyChip } from '../resource-properties-form/property-chip';

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface CreateCollectionPropertiesListDataProps {
    properties: CollectionProperties;
}

interface CreateCollectionPropertiesListActionProps {
    handleDelete: (key: string, value: string) => void;
}

const mapStateToProps = (state: RootState): CreateCollectionPropertiesListDataProps => {
    const properties = COLLECTION_CREATE_FORM_SELECTOR(state, 'properties');
    return { properties };
};

const mapDispatchToProps = (dispatch: Dispatch): CreateCollectionPropertiesListActionProps => ({
    handleDelete: (key: string, value: string) => dispatch<any>(removePropertyFromCreateCollectionForm(key, value))
});

type CreateCollectionPropertiesListProps = CreateCollectionPropertiesListDataProps &
    CreateCollectionPropertiesListActionProps & WithStyles<CssRules>;

const List = withStyles(styles)(
    ({ classes, handleDelete, properties }: CreateCollectionPropertiesListProps) =>
        <div>
            {properties &&
                Object.keys(properties).map(k =>
                    Array.isArray(properties[k])
                    ? (properties[k] as string[]).map((v: string) =>
                        getPropertyChip(
                            k, v,
                            () => handleDelete(k, v),
                            classes.tag))
                    : getPropertyChip(
                        k, (properties[k] as string),
                        () => handleDelete(k, (properties[k] as string)),
                        classes.tag))
                }
        </div>
);

export const CreateCollectionPropertiesList = connect(mapStateToProps, mapDispatchToProps)(List);