// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { Collapse, Typography } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { CustomStyleRulesCallback } from 'common/custom-theme';

type CssRules = 'description' | 'fadedDescription';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    description: {
        paddingBottom: '-1rem',
    },
    fadedDescription: {
        position: 'relative',
        WebkitMaskImage: 'linear-gradient(to bottom, black 18rem, transparent 20rem)',
        maskImage: 'linear-gradient(to bottom, black 18rem, transparent 20rem)',
        WebkitMaskSize: '100% 100%',
        maskSize: '100% 100%',
        WebkitMaskRepeat: 'no-repeat',
        maskRepeat: 'no-repeat',
    },
});

type CollapsibleDescriptionProps = {
    description: string;
    showDescription: boolean;
    collapsedSize?: string;
};

export const CollapsibleDescription = withStyles(styles)((props: CollapsibleDescriptionProps & WithStyles<CssRules>) => {
    const { classes, description, showDescription, collapsedSize } = props;
    const [fadeDescription, setFadeDescription] = React.useState(!showDescription);

    //prevents jarring pop-in/out animations
    React.useEffect(() => {
        setTimeout(() => setFadeDescription(!fadeDescription), 80);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [showDescription]);

    return (
        <section>
            <Collapse
                in={showDescription}
                timeout='auto'
                collapsedSize={collapsedSize || '2.3rem'}
            >
                <section data-cy='resource-description'>
                    <Typography
                        className={classNames(fadeDescription ? classes.description : classes.fadedDescription)}
                        component='div'
                        //dangerouslySetInnerHTML is ok here only if description is sanitized,
                        //which it is before it is loaded into the redux store
                        dangerouslySetInnerHTML={{ __html: description }}
                    />
                </section>
            </Collapse>
        </section>
    );
});