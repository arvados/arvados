// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
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
    tailUuid: string;
    headUuid: string;
    properties: any;
}

interface MetadataProps {
    items: MetadataTable[];
    uuid: string;
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
            {props.items.map((it, index) =>
                <TableRow key={index}>
                    <TableCell className={props.classes.cell}>{it.uuid}</TableCell>
                    <TableCell className={props.classes.cell}>{it.linkClass}</TableCell>
                    <TableCell className={props.classes.cell}>{it.name}</TableCell>
                    <TableCell className={props.classes.cell}>{it.tailUuid}</TableCell>
                    <TableCell className={props.classes.cell}>{it.headUuid === props.uuid ? 'this' : it.headUuid}</TableCell>
                    <TableCell className={props.classes.cell}>{JSON.stringify(it.properties)}</TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>
);