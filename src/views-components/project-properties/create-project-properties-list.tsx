// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import {
    withStyles,
    StyleRulesCallback,
    WithStyles,
} from '@material-ui/core';
import { RootState } from '~/store/store';
import { removePropertyFromCreateProjectForm, PROJECT_CREATE_FORM_SELECTOR, ProjectProperties } from '~/store/projects/project-create-actions';
import { ArvadosTheme } from '~/common/custom-theme';
import { getPropertyChip } from '../resource-properties-form/property-chip';

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface CreateProjectPropertiesListDataProps {
    properties: ProjectProperties;
}

interface CreateProjectPropertiesListActionProps {
    handleDelete: (key: string, value: string) => void;
}

const mapStateToProps = (state: RootState): CreateProjectPropertiesListDataProps => {
    const properties = PROJECT_CREATE_FORM_SELECTOR(state, 'properties');
    return { properties };
};

const mapDispatchToProps = (dispatch: Dispatch): CreateProjectPropertiesListActionProps => ({
    handleDelete: (key: string, value: string) => dispatch<any>(removePropertyFromCreateProjectForm(key, value))
});

type CreateProjectPropertiesListProps = CreateProjectPropertiesListDataProps &
    CreateProjectPropertiesListActionProps & WithStyles<CssRules>;

const List = withStyles(styles)(
    ({ classes, handleDelete, properties }: CreateProjectPropertiesListProps) =>
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

export const CreateProjectPropertiesList = connect(mapStateToProps, mapDispatchToProps)(List);