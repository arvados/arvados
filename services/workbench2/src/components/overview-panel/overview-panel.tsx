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
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';
import { CollapsibleDescription } from 'components/collapsible-description/collapsible-description';
import { CollectionResource } from 'models/collection';
import { ProjectResource } from 'models/project';

type CssRules = 'root' | 'tag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        padding: theme.spacing(1),
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5),
    },
});

type OverviewPanelProps = {
    resource: ProjectResource | CollectionResource | undefined;
    detailsElement: React.ReactNode;
} & WithStyles<CssRules>;

const mapStateToProps = (state: RootState): Pick<OverviewPanelProps, 'resource'> => {
    return {
        resource: getResource<any>(state.properties.currentRouteUuid)(state.resources),
    };
};

export const OverviewPanel = connect(mapStateToProps)(withStyles(styles)((({ resource, detailsElement, classes }: OverviewPanelProps) => {
    if (!resource) {
        return null;
    }

    const hasDescription = resource.description && resource.description.length > 0;
    const hasProperties = (typeof resource.properties === 'object' && Object.keys(resource.properties).length > 0);

    const [showDescription, setShowDescription] = useState(false);

    return (
        <section className={classes.root}>
            <Grid item xs={12} md={12}>
                <DetailsAttribute
                    label={'Description'}
                    button={hasDescription
                                ? <ExpandChevronRight expanded={showDescription} onClick={() => setShowDescription(!showDescription)} />
                                : undefined}>
                    {hasDescription
                        ? <CollapsibleDescription description={resource.description} showDescription={showDescription} />
                        : <Typography>No description available</Typography>}
                </DetailsAttribute>
                {detailsElement}
            </Grid>
            <section>
                {hasProperties &&
                    <>
                        <DetailsAttribute label='Properties' />
                        <section>
                            {Object.keys(resource.properties).map((k) =>
                                Array.isArray(resource.properties[k])
                                ? resource.properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                                : getPropertyChip(k, resource.properties[k], undefined, classes.tag)
                            )}
                        </section>
                    </>
                }
            </section>
        </section>
    );
})));
