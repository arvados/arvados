// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import { connect } from 'react-redux';
import { Grid, Typography } from '@mui/material';
import { RootState } from 'store/store';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { ArvadosTheme } from 'common/custom-theme';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { getResource } from 'store/resources/resources';
import { ResourceKind } from 'models/resource';
import { resourceLabel } from 'common/labels';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { ResourceWithName } from 'views-components/data-explorer/renderers';
import { GroupClass } from 'models/group';
import { formatDate } from 'common/formatters';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { CollapsibleDescription } from 'components/collapsible-description/collapsible-description';

type CssRules = 'root' | 'tag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        padding: theme.spacing(1),
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

export const ProjectOverview = connect(mapStateToProps)(withStyles(styles)((({ project, classes }: ProjectOverviewProps) => {
    if (!project || project.kind !== ResourceKind.PROJECT) {
        return null;
    }
    const hasDescription = project.description && project.description.length > 0;
    const hasProperties = (typeof project.properties === 'object' && Object.keys(project.properties).length > 0);

    const [showDescription, setShowDescription] = useState(false);

    return (
        <Grid container spacing={1} className={classes.root}>
            <Grid item xs={12} md={12} onClick={() => setShowDescription(!showDescription)}>
                <DetailsAttribute label={'Description'} button={hasDescription ? <ExpandChevronRight expanded={showDescription} /> : undefined}>
                    {hasDescription
                        ? <CollapsibleDescription description={project.description} showDescription={showDescription} />
                        : <Typography>No description available</Typography>}
                </DetailsAttribute>
            </Grid>
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
                    value={formatDate(project.createdAt)}
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Last modified'
                    value={formatDate(project.modifiedAt)}
                />
            </Grid>
            <Grid item xs={12} md={6}>
                <DetailsAttribute
                    label='Last modified by'
                    linkToUuid={project.modifiedByUserUuid}
                    uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />}
                    />
            </Grid>
            {hasProperties &&
            <>
                <DetailsAttribute label='Properties' />
                <Grid item xs={12} md={12}>
                    {Object.keys(project.properties).map((k) =>
                        Array.isArray(project.properties[k])
                            ? project.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                            : getPropertyChip(k, project.properties[k], undefined, classes.tag)
                    )}
                </Grid>
            </>
            }
        </Grid>
    );
})));
