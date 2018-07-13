// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Typography from '@material-ui/core/Typography';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '../../common/custom-theme';

interface AttributeDataProps {
    label: string;
    value?: string | number;
    link?: string;
}

type AttributeProps = AttributeDataProps & WithStyles<CssRules>;

class Attribute extends React.Component<AttributeProps> {

    hasLink() {
        return !!this.props.link;
    }

    render() {
        const { label, link, value, children, classes } = this.props;
        return <Typography component="div" className={classes.attribute}>
                    <Typography component="span" className={classes.label}>{label}</Typography>
                    { this.hasLink() ? (
                        <a href='{link}' className={classes.link} target='_blank'>{value}</a>
                    ) : (
                        <Typography component="span" className={classes.value}>
                            {value}
                            {children}
                        </Typography>
                    )}
                </Typography>;
    }

}

type CssRules = 'attribute' | 'label' | 'value' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    attribute: {
        display: 'flex',
        alignItems: 'flex-start',
        marginBottom: theme.spacing.unit
    },
    label: {
        color: theme.palette.grey["500"],
        width: '40%'
    },
    value: {
        width: '60%',
        display: 'flex',
        alignItems: 'flex-start',
        textTransform: 'capitalize'
    },
    link: {
        color: theme.palette.primary.main,
        textDecoration: 'none'
    }
});

export default withStyles(styles)(Attribute);