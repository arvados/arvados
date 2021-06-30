// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { DefaultViewDataProps, DefaultView } from 'components/default-view/default-view';
import { ArvadosTheme } from 'common/custom-theme';
import { DetailsIcon } from 'components/icon/icon';

type CssRules = 'classRoot';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    classRoot: {
        marginTop: theme.spacing.unit * 4,
        marginBottom: theme.spacing.unit * 4,
    },
});
type DataTableDefaultViewDataProps = Partial<Pick<DefaultViewDataProps, 'icon' | 'messages'>>;
type DataTableDefaultViewProps = DataTableDefaultViewDataProps & WithStyles<CssRules>;

export const DataTableDefaultView = withStyles(styles)(
    ({ classes, ...props }: DataTableDefaultViewProps) => {
        const icon = props.icon || DetailsIcon;
        const messages = props.messages || ['No items found'];
        return <DefaultView {...classes} {...{ icon, messages }} />;
    });
