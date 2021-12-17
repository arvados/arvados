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
    PROJECT_UPDATE_FORM_SELECTOR,
    PROJECT_UPDATE_FORM_NAME,
} from 'store/projects/project-update-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { getPropertyChip } from '../resource-properties-form/property-chip';
import { ProjectProperties } from 'store/projects/project-create-actions';
import { removePropertyFromResourceForm } from 'store/resources/resources-actions';

type CssRules = 'tag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    }
});

interface UpdateProjectPropertiesListDataProps {
    properties: ProjectProperties;
}

interface UpdateProjectPropertiesListActionProps {
    handleDelete: (key: string, value: string) => void;
}

const mapStateToProps = (state: RootState): UpdateProjectPropertiesListDataProps => {
    const properties = PROJECT_UPDATE_FORM_SELECTOR(state, 'properties');
    return { properties };
};

const mapDispatchToProps = (dispatch: Dispatch): UpdateProjectPropertiesListActionProps => ({
    handleDelete: (key: string, value: string) => dispatch<any>(removePropertyFromResourceForm(key, value, PROJECT_UPDATE_FORM_NAME))
});

type UpdateProjectPropertiesListProps = UpdateProjectPropertiesListDataProps &
    UpdateProjectPropertiesListActionProps & WithStyles<CssRules>;

const List = withStyles(styles)(
    ({ classes, handleDelete, properties }: UpdateProjectPropertiesListProps) =>
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

export const UpdateProjectPropertiesList = connect(mapStateToProps, mapDispatchToProps)(List);