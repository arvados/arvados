// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ArvadosTheme, CustomStyleRulesCallback } from 'common/custom-theme';
import withStyles, { WithStyles } from '@mui/styles/withStyles';
import { Typography, Grid } from '@mui/material';
import { ProjectResource } from 'models/project';
import { CollectionResource } from 'models/collection';
import { WorkflowResource } from 'models/workflow';
import { ContainerRequestResource } from 'models/container-request';

type CssRules =
    | 'descriptionPreview'
    | 'descriptionPreviewMore';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    descriptionPreview: {
        position: 'relative',
        margin: '0 .7rem',
        // Max height 2 lines + 1.5 margin
        maxHeight: 'calc(0.875rem * 2 + 0.75rem)',
        overflow: 'hidden',
        // All text small and inline
        '& :is(h1, h2, h3, h4, h5, h6, p)': {
            display: 'inline',
            fontSize: '0.875rem',
        },
        // Return line breaks after paragraphs
        '& p': {
            '&::after': {
                content: `""`,
                display: 'block',
            },
        },
        // Add line breaks before images to avoid pushing text away
        // Conveniently, the editor wraps images in paragraphs
        '& p:has(> img)::before': {
            content: `""`,
            display: 'block',
        },
        // Headers bold
        '& :is(h1, h2, h3, h4, h5, h6)': {
            fontWeight: 'bold',
        },
        // Header separator - this style doesn't work when nested for some reason
        '& :is(h1, h2, h3, h4, h5, h6)::after': {
            fontWeight: 'bold',
            content: `" —"`,
        },
        //Add small fade out to bottom
        '&::before': {
            content: `""`,
            width: '100%',
            height: 'calc(0.875rem + 0.5rem)',
            position: 'absolute',
            left: 0,
            bottom: 0,
            background: 'linear-gradient(transparent 0, #fff)',
        },
    },
    descriptionPreviewMore: {
        cursor: "pointer",
        color: '#017ead',
        margin: '0.25rem .7rem',
    },
});

interface DescriptionPreviewDataProps {
    resource: ProjectResource | CollectionResource | WorkflowResource | ContainerRequestResource;
};

type DescriptionPreviewProps = WithStyles<CssRules> & DescriptionPreviewDataProps;

export const DescriptionPreview =
    withStyles(styles)((props: DescriptionPreviewProps) => {
        const { classes, resource } = props;

        return (
            <Grid>
                <Typography
                    className={classes.descriptionPreview}
                    component="div"
                    //dangerouslySetInnerHTML is ok here only if description is sanitized,
                    //which it is before it is loaded into the redux store
                    dangerouslySetInnerHTML={{
                        __html: resource.description,
                    }}
                />
                <Typography
                    className={classes.descriptionPreviewMore}
                    onClick={() => {}}
                >
                    Read more...
                </Typography>
            </Grid>
        );
    });
