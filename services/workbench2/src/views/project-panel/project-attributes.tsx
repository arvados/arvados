// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Grid } from '@mui/material';
import { RootState } from 'store/store';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { ArvadosTheme } from 'common/custom-theme';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { getResource } from 'store/resources/resources';
import { ResourceKind } from 'models/resource';
import { resourceLabel } from 'common/labels';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { ResourceWithName } from 'views-components/data-explorer/renderers';
import { GroupClass } from 'models/group';
import { formatDateTime } from 'common/formatters';

type CssRules = 'root' | 'tag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5),
    },
});

type ProjectOverviewProps = {
    project: any;
} & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): Pick<ProjectOverviewProps, 'project'> => {
    return {
        project: getResource(state.properties.projectPanelCurrentUuid)(state.resources),
    };
};

export const ProjectAttributes = connect(mapStateToProps)(withStyles(styles)((({ project, classes }: ProjectOverviewProps) => {
    if (!project || project.kind !== ResourceKind.PROJECT) {
        return null;
    }
    return (
        <Grid container spacing={1} className={classes.root}>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Type'
                    value={project.groupClass === GroupClass.FILTER ? 'Filter group' : resourceLabel(ResourceKind.PROJECT)}
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='UUID'
                    linkToUuid={project.uuid}
                    value={project.uuid}
                    />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Owner'
                    linkToUuid={project.ownerUuid}
                    uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />}
                    />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Created at'
                    value={formatDateTime(project.createdAt)}
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Last modified'
                    value={formatDateTime(project.modifiedAt)}
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Last modified by'
                    linkToUuid={project.modifiedByUserUuid}
                    uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />}
                    />
            </Grid>
        </Grid>
    );
})));
