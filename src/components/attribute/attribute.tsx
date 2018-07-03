// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Typography from '@material-ui/core/Typography';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'src/common/custom-theme';

interface AttributeDataProps {
    label: string;
}

type AttributeProps = AttributeDataProps & WithStyles<CssRules>;

class Attribute extends React.Component<AttributeProps> {

    render() {
        const { label, children, classes } = this.props;
        return <Typography component="div" className={classes.attribute}>
                <span className={classes.label}>{label}</span>
                <span className={classes.value}>{children}</span>
            </Typography>;
    }

}

type CssRules = 'attribute' | 'label' | 'value';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    attribute: {
        display: 'flex',
        alignItems: 'center',
        marginBottom: theme.spacing.unit
    },
    label: {
        color: theme.palette.grey["500"],
        width: '40%'
    },
    value: {
        display: 'flex',
        alignItems: 'center'
    }
});

export default withStyles(styles)(Attribute);