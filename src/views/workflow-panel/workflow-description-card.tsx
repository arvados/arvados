// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles, Card, CardHeader, Typography } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { WorkflowIcon } from '~/components/icon/icon';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';

export type CssRules = 'card';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    }
});

interface WorkflowDescriptionCardDataProps {
}

type WorkflowDescriptionCardProps = WorkflowDescriptionCardDataProps & WithStyles<CssRules>;

export const WorkflowDescriptionCard = withStyles(styles)(
    ({ classes }: WorkflowDescriptionCardProps) => {
        return <Card className={classes.card}>
            <CardHeader
                title={<Typography noWrap variant="body2">
                    Workflow description:
                </Typography>} />
            <DataTableDefaultView
                icon={WorkflowIcon}
                messages={['Please select a workflow to see its description.']} />
        </Card>;
    });