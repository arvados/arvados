// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from './process-information-card';
import { DefaultView } from '~/components/default-view/default-view';
import { ProcessIcon } from '~/components/icon/icon';
import { Process } from '~/store/processes/process';

export interface ProcessPanelRootDataProps {
    process?: Process;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps;

export const ProcessPanelRoot = (props: ProcessPanelRootProps) =>
    props.process
        ? <Grid container>
            <Grid item xs={7}>
                <ProcessInformationCard
                    process={props.process}
                    onContextMenu={props.onContextMenu} />
            </Grid>
        </Grid>
        : <Grid container
            alignItems='center'
            justify='center'>
            <DefaultView
                icon={ProcessIcon}
                messages={['Process not found']} />
        </Grid>;
