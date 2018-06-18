// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Theme } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from "@material-ui/core/styles";
import Paper from "@material-ui/core/Paper/Paper";
import Table from "@material-ui/core/Table/Table";
import TableHead from "@material-ui/core/TableHead/TableHead";
import TableRow from "@material-ui/core/TableRow/TableRow";
import TableCell from "@material-ui/core/TableCell/TableCell";
import TableBody from "@material-ui/core/TableBody/TableBody";

type CssRules = 'root' | 'table';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    root: {
        width: '100%',
        marginTop: theme.spacing.unit * 3,
        overflowX: 'auto',
    },
    table: {
        minWidth: 700,
    },
});

interface ProjectListProps {
}

class ProjectList extends React.Component<ProjectListProps & WithStyles<CssRules>, {}> {
    render() {
        const {classes} = this.props;
        return <Paper className={classes.root}>
            <Table className={classes.table}>
                <TableHead>
                    <TableRow>
                        <TableCell>Name</TableCell>
                        <TableCell>Status</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Shared by</TableCell>
                        <TableCell>File size</TableCell>
                        <TableCell>Last modified</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    <TableRow>
                        <TableCell>Project 1</TableCell>
                        <TableCell>Complete</TableCell>
                        <TableCell>Project</TableCell>
                        <TableCell>John Doe</TableCell>
                        <TableCell>1.5 GB</TableCell>
                        <TableCell>9:22 PM</TableCell>
                    </TableRow>
                </TableBody>
            </Table>
        </Paper>;
    }
}

export default withStyles(styles)(ProjectList);
