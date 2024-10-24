// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ChevronRight } from '@mui/icons-material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { IconButton } from "@mui/material";

type CssRules = 'root' | 'default' | 'expanded';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '24px',
        height: '24px',
        cursor: 'pointer',
        marginLeft: '1em',
        marginTop: "-.2em",
    },
    default: {
        transition: 'all 0.1s ease',
        transform: 'rotate(0deg)',
    },
    expanded: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    },
});

export interface ExpandChevronRightDataProps {
    expanded: boolean;
}

type ExpandChevronRightProps = ExpandChevronRightDataProps & WithStyles<CssRules>;

export const ExpandChevronRight = withStyles(styles)(
    class extends React.Component<ExpandChevronRightProps, {}> {
        render() {
            const { classes, expanded } = this.props;
            return (
                <IconButton className={classes.root}>
                    <ChevronRight className={expanded ? classes.expanded : classes.default} />
                </IconButton>
            );
        }
    }
);
