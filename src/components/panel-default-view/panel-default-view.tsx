// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { DefaultViewDataProps, DefaultView } from 'components/default-view/default-view';

type CssRules = 'classRoot' | 'classIcon' | 'classMessage';

const styles: StyleRulesCallback<CssRules> = () => ({
    classRoot: {
        position: 'absolute',
        width: '80%',
        left: '50%',
        top: '50%',
        transform: 'translate(-50%, -50%)'
    },
    classMessage: {
        fontSize: '1.75rem',
    },
    classIcon: {
        fontSize: '6rem'
    }
});

type PanelDefaultViewProps = Pick<DefaultViewDataProps, 'icon' | 'messages'> & WithStyles<CssRules>;

export const PanelDefaultView = withStyles(styles)(
    ({ classes, ...props }: PanelDefaultViewProps) =>
        <DefaultView {...classes} {...props} />);
