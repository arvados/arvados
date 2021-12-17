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
import { ArvadosTheme } from 'common/custom-theme';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { removePropertyFromResourceForm } from 'store/resources/resources-actions';
import { formValueSelector } from 'redux-form';

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface ResourcePropertiesListDataProps {
    properties: {[key: string]: string | string[]};
}

interface ResourcePropertiesListActionProps {
    handleDelete: (key: string, value: string) => void;
}

type ResourcePropertiesListProps = ResourcePropertiesListDataProps &
ResourcePropertiesListActionProps & WithStyles<CssRules>;

const List = withStyles(styles)(
    ({ classes, handleDelete, properties }: ResourcePropertiesListProps) =>
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

export const resourcePropertiesList = (formName: string) =>
    connect(
        (state: RootState): ResourcePropertiesListDataProps => ({
            properties: formValueSelector(formName)(state, 'properties')
        }),
        (dispatch: Dispatch): ResourcePropertiesListActionProps => ({
                handleDelete: (key: string, value: string) => dispatch<any>(removePropertyFromResourceForm(key, value, formName))
        })
    )(List);