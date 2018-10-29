// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Table, TableHead, TableCell, TableRow, TableBody, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type CssRules = 'cell';

const styles: StyleRulesCallback<CssRules> = theme => ({
    cell: {
        paddingRight: theme.spacing.unit * 2
    }
});

interface MetadataTable {
    uuid: string;
    linkClass: string;
    name: string;
    tail: string;
    head: string;
    properties: any;
}

interface MetadataProps {
    items: MetadataTable[];
}


export const MetadataTab = withStyles(styles)((props: MetadataProps & WithStyles<CssRules>) =>
    <Table>
        <TableHead>
            <TableRow>
                <TableCell>uuid</TableCell>
                <TableCell>link_class</TableCell>
                <TableCell>name</TableCell>
                <TableCell>tail</TableCell>
                <TableCell>head</TableCell>
                <TableCell>properties</TableCell>
            </TableRow>
        </TableHead>
        <TableBody>
            {props.items.map((it: any, index: number) => {
                return (
                    <TableRow key={index}>
                        {tableCell(it.uuid, props.classes)}
                        {tableCell(it.linkClass, props.classes)}
                        {tableCell(it.name, props.classes)}
                        {tableCell(it.tailUuid, props.classes)}
                        {tableCell(it.headUuid, props.classes)}
                        {tableCell(JSON.stringify(it.properties, null, 2), props.classes)}
                    </TableRow>
                );
            })}
        </TableBody>
    </Table>
);

const tableCell = (value: string, classes: any) =>
    <TableCell className={classes.cell}>{value}</TableCell>;