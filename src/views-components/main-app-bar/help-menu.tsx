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

enum helpMenuTitles {
    PIPELINES_DATASETS = "Public Pipelines and Data sets",
    TUTORIALS = "Tutorials and User guide",
    API_REFERENCE = "API Reference",
    SDK_REFERENCE = "SDK Reference"
}

const links = [
    {
        title: helpMenuTitles.PIPELINES_DATASETS,
        link: helpMenuLinks.PIPELINES_DATASETS
    },
    {
        title: helpMenuTitles.TUTORIALS,
        link: helpMenuLinks.TUTORIALS
    },
    {
        title: helpMenuTitles.API_REFERENCE,
        link: helpMenuLinks.API_REFERENCE
    },
    {
        title: helpMenuTitles.SDK_REFERENCE,
        link: helpMenuLinks.SDK_REFERENCE
    },
];

export const HelpMenu = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) =>
        <DropdownMenu
            icon={<HelpIcon />}
            id="help-menu"
            title="Help">
            <Typography variant="body1" className={classes.title}>Help</Typography>
            {
                links.map(link =>
                <a key={link.title} href={link.link} target="_blank" className={classes.link}>
                    <MenuItem>
                        <HelpIcon className={classes.icon} />
                        <Typography variant="body1" className={classes.linkTitle}>{link.title}</Typography>
                    </MenuItem>
                </a>)
            }
        </DropdownMenu>
);
