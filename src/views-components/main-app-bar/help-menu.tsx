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
        paddingBottom: theme.spacing.unit * 0.5,
        outline: 'none',
    },
    linkTitle: {
        marginLeft: theme.spacing.unit
    }
});

const links = [
    {
        title: "Public Pipelines and Data sets",
        link: "https://dev.arvados.org/projects/arvados/wiki/Public_Pipelines_and_Datasets",
    },
    {
        title: "Tutorials and User guide",
        link: "http://doc.arvados.org/user/",
    },
    {
        title: "API Reference",
        link: "http://doc.arvados.org/api/",
    },
    {
        title: "SDK Reference",
        link: "http://doc.arvados.org/sdk/"
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
