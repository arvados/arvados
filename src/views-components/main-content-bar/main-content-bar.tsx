// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Toolbar, IconButton, Tooltip, Grid } from "@material-ui/core";
import { DetailsIcon } from "~/components/icon/icon";
import { Breadcrumbs } from "~/views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { matchWorkflowRoute, matchSshKeysRoute, matchRepositoriesRoute, matchVirtualMachineRoute, matchKeepServicesRoute } from '~/routes/routes';
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';

interface MainContentBarProps {
    onDetailsPanelToggle: () => void;
    buttonVisible: boolean;
}

const isButtonVisible = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    return !matchWorkflowRoute(pathname) && !matchVirtualMachineRoute(pathname) &&
        !matchRepositoriesRoute(pathname) && !matchSshKeysRoute(pathname) && !matchKeepServicesRoute(pathname);
};

export const MainContentBar = connect((state: RootState) => ({
    buttonVisible: isButtonVisible(state)
}), {
        onDetailsPanelToggle: toggleDetailsPanel
    })((props: MainContentBarProps) =>
        <Toolbar>
            <Grid container>
                <Grid container item xs alignItems="center">
                    <Breadcrumbs />
                </Grid>
                <Grid item>
                    {props.buttonVisible && <Tooltip title="Additional Info">
                        <IconButton color="inherit" onClick={props.onDetailsPanelToggle}>
                            <DetailsIcon />
                        </IconButton>
                    </Tooltip>}
                </Grid>
            </Grid>
        </Toolbar>);
