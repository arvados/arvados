// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Toolbar, IconButton, Tooltip, Grid } from "@material-ui/core";
import { DetailsIcon } from "~/components/icon/icon";
import { Breadcrumbs } from "~/views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import * as Routes from '~/routes/routes';
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';

interface MainContentBarProps {
    onDetailsPanelToggle: () => void;
    buttonVisible: boolean;
}

const isButtonVisible = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    return !Routes.matchWorkflowRoute(pathname) && !Routes.matchUserVirtualMachineRoute(pathname) &&
        !Routes.matchAdminVirtualMachineRoute(pathname) && !Routes.matchRepositoriesRoute(pathname) &&
        !Routes.matchSshKeysAdminRoute(pathname) && !Routes.matchSshKeysUserRoute(pathname) &&
        !Routes.matchSiteManagerRoute(pathname) &&
        !Routes.matchKeepServicesRoute(pathname) && !Routes.matchComputeNodesRoute(pathname) &&
        !Routes.matchApiClientAuthorizationsRoute(pathname) && !Routes.matchUsersRoute(pathname) &&
        !Routes.matchMyAccountRoute(pathname) && !Routes.matchLinksRoute(pathname);
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
