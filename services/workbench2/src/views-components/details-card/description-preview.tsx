// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { ArvadosTheme, CustomStyleRulesCallback } from 'common/custom-theme';
import withStyles, { WithStyles } from '@mui/styles/withStyles';
import { Typography, Grid } from '@mui/material';
import descriptionDialogActions from 'store/description-dialog/description-dialog-actions';
import { connect } from 'react-redux';
import { ProjectResource } from 'models/project';
import { CollectionResource } from 'models/collection';
import { WorkflowResource } from 'models/workflow';
import { ContainerRequestResource } from 'models/container-request';

type CssRules =
    | 'wrapper'
    | 'preview'
    | 'overflowButton';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    wrapper: {
        // Max height 3 lines at line height 1.5
        // Updates to maxHeight must be mirrored in overflowButton bottom
        maxHeight: 'calc(0.875rem * 1.5 * 3)',
        overflow: 'hidden',
        position: 'relative',
        // Added bottom margin to match space above title
        margin: '0 0 8px',
    },
    preview: {
        margin: '0 1rem',
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
    },
    overflowButton: {
        cursor: 'pointer',
        // Avoid taking up more space than necessary
        lineHeight: 1,
        // Use contentbox so that vertical text alignment is nice
        // and to easily add padding to the height
        boxSizing: 'content-box',
        // Must use calc to account for margin due to content-box
        width: 'calc(100% - (0.7rem * 2))',
        // Start height at font size
        height: 'calc(0.875rem)',
        position: 'absolute',
        // Bottom calc must match wrapper maxHeight
        bottom: 'calc((100% - calc(0.875rem * 1.5 * 3)) * 10000)',
        color: theme.palette.primary.main,
        margin: '0 1rem',
        // Added padding for overlapping linear gradient
        // Bottom padding instead of margin prevents covered content from peeking
        padding: 'calc(0.875rem * 1.5) 0 0.25rem',
        // Gradient end should match line height
        background: 'linear-gradient(transparent 0rem, #fff 0.875rem)',
    },
});

interface DescriptionPreviewDispatchProps {
    openDescriptionDialog: (uuid: string) => void;
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openDescriptionDialog: (uuid: string) => {
        dispatch<any>(descriptionDialogActions.openDialog(uuid));
    },
});

interface DescriptionPreviewDataProps {
    resource: ProjectResource | CollectionResource | WorkflowResource | ContainerRequestResource;
};

type DescriptionPreviewProps = WithStyles<CssRules> & DescriptionPreviewDispatchProps & DescriptionPreviewDataProps;

export const DescriptionPreview = connect(
    null,
    mapDispatchToProps
)(
    withStyles(styles)((props: DescriptionPreviewProps) => {
        const { classes, resource } = props;

        return resource.description?.length ? (
            <Grid className={classes.wrapper}>
                <Typography
                    className={classes.preview}
                    component="div"
                    //dangerouslySetInnerHTML is ok here only if description is sanitized,
                    //which it is before it is loaded into the redux store
                    dangerouslySetInnerHTML={{
                        __html: resource.description,
                    }}
                />
                <Typography
                    className={classes.overflowButton}
                    onClick={() => {
                        props.openDescriptionDialog(resource.uuid);
                    }}
                >
                    Read more...
                </Typography>
            </Grid>
        ) : <></>;
    })
);
