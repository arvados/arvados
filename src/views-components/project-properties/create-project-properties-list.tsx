// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { withStyles, StyleRulesCallback, WithStyles, Chip } from '@material-ui/core';
import { RootState } from '~/store/store';
import { removePropertyFromCreateProjectForm, PROJECT_CREATE_FORM_SELECTOR, ProjectProperties } from '~/store/projects/project-create-actions';
import { ArvadosTheme } from '~/common/custom-theme';

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
    handleDelete: (key: string) => void;
}

const mapStateToProps = (state: RootState): CreateProjectPropertiesListDataProps => {
    const properties = PROJECT_CREATE_FORM_SELECTOR(state, 'properties');
    return { properties };
};

const mapDispatchToProps = (dispatch: Dispatch): CreateProjectPropertiesListActionProps => ({
    handleDelete: (key: string) => dispatch<any>(removePropertyFromCreateProjectForm(key))
});

type CreateProjectPropertiesListProps = CreateProjectPropertiesListDataProps & 
    CreateProjectPropertiesListActionProps & WithStyles<CssRules>;

const List = withStyles(styles)(
    ({ classes, handleDelete, properties }: CreateProjectPropertiesListProps) =>
        <div>
            {properties &&
                Object.keys(properties).map(k => {
                    return <Chip key={k} className={classes.tag}
                        onDelete={() => handleDelete(k)}
                        label={`${k}: ${properties[k]}`} />;
                })}
        </div>
);

export const CreateProjectPropertiesList = connect(mapStateToProps, mapDispatchToProps)(List);