// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { MenuItem, Typography } from "@material-ui/core";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { HelpIcon } from "~/components/icon/icon";
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';

type CssRules = 'link' | 'icon' | 'title' | 'linkTitle';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    },
    icon: {
        width: '16px',
        height: '16px'
    },
    title: {
        marginLeft: theme.spacing.unit * 2,
        marginBottom: theme.spacing.unit * 0.5,
        outline: 'none',
    },
    linkTitle: {
        marginLeft: theme.spacing.unit
    }
});

enum helpMenuLinks {
    PIPELINES_DATASETS = "https://dev.arvados.org/projects/arvados/wiki/Public_Pipelines_and_Datasets",
    TUTORIALS = "http://doc.arvados.org/user/",
    API_REFERENCE = "http://doc.arvados.org/api/",
    SDK_REFERENCE = "http://doc.arvados.org/sdk/"
}

export const HelpMenu = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) =>
        <DropdownMenu
            icon={<HelpIcon />}
            id="help-menu"
            title="Help">
            <Typography variant="body1" className={classes.title}>Help</Typography>
            {menuItem("Public Pipelines and Data sets", helpMenuLinks.PIPELINES_DATASETS, classes)}
            {menuItem("Tutorials and User guide", helpMenuLinks.TUTORIALS, classes)}
            {menuItem("API Reference", helpMenuLinks.API_REFERENCE, classes)}
            {menuItem("SDK Reference", helpMenuLinks.SDK_REFERENCE, classes)}
        </DropdownMenu>
);

    // Todo: change help icon
const menuItem = (title: string, link: string, classes: Record<CssRules, string>) =>
    <a href={link} target="_blank" className={classes.link}>
        <MenuItem>
            <HelpIcon className={classes.icon} />
            <Typography variant="body1" className={classes.linkTitle}>{title}</Typography>
        </MenuItem>
    </a>;
