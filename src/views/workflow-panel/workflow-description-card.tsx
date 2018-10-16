// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    CardContent,
    Tab,
    Tabs,
    Table,
    TableHead,
    TableCell,
    TableBody,
    TableRow,
    Divider
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { WorkflowIcon } from '~/components/icon/icon';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { WorkflowResource, parseWorkflowDefinition, getWorkflowInputs, getInputLabel, stringifyInputType } from '~/models/workflow';
import { WorkflowGraph } from "~/views/workflow-panel/workflow-graph";

export type CssRules = 'root' | 'tab' | 'inputTab';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        height: '100%'
    },
    tab: {
        minWidth: '33%'
    },
    inputTab: {
        height: 'calc(100% - 48px - 16px * 2)',
        overflowX: 'auto',
        overflowY: 'hidden',
        '&:last-child': {
            paddingBottom: theme.spacing.unit / 2,
        }
    }
});

interface WorkflowDetailsCardDataProps {
    workflow?: WorkflowResource;
}

type WorkflowDetailsCardProps = WorkflowDetailsCardDataProps & WithStyles<CssRules>;

export const WorkflowDetailsCard = withStyles(styles)(
    class extends React.Component<WorkflowDetailsCardProps> {
        state = {
            value: 0,
        };

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }

        render() {
            const { classes, workflow } = this.props;
            const { value } = this.state;
            return <div className={classes.root}>
                <Tabs value={value} onChange={this.handleChange} centered={true}>
                    <Tab className={classes.tab} label="Description" />
                    <Tab className={classes.tab} label="Inputs" />
                    <Tab className={classes.tab} label="Graph" />
                </Tabs>
                {value === 0 && <CardContent>
                    {workflow ? <div>
                        {workflow.description}
                    </div> : (
                            <DataTableDefaultView
                                icon={WorkflowIcon}
                                messages={['Please select a workflow to see its description.']} />
                        )}
                </CardContent>}
                {value === 1 && <CardContent className={classes.inputTab}>
                    {workflow
                        ? this.renderInputsTable()
                        : <DataTableDefaultView
                            icon={WorkflowIcon}
                            messages={['Please select a workflow to see its inputs.']} />
                    }
                </CardContent>}
                {value === 2 && <CardContent className={classes.inputTab}>
                    {workflow
                        ? <WorkflowGraph workflow={workflow}/>
                        : <DataTableDefaultView
                            icon={WorkflowIcon}
                            messages={['Please select a workflow to see its visualisation.']} />
                    }
                </CardContent>}
            </div>;
        }

        get inputs() {
            if (this.props.workflow) {
                const definition = parseWorkflowDefinition(this.props.workflow);
                if (definition) {
                    return getWorkflowInputs(definition);
                }
            }
            return;
        }

        renderInputsTable() {
            return <Table>
                <TableHead>
                    <TableRow>
                        <TableCell>Label</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Description</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {this.inputs && this.inputs.map(input =>
                        <TableRow key={input.id}>
                            <TableCell>{getInputLabel(input)}</TableCell>
                            <TableCell>{stringifyInputType(input)}</TableCell>
                            <TableCell>{input.doc}</TableCell>
                        </TableRow>)}
                </TableBody>
            </Table>;
        }
    });
