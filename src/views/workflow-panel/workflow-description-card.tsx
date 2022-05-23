// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
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
    Grid,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { WorkflowIcon } from 'components/icon/icon';
import { DataTableDefaultView } from 'components/data-table-default-view/data-table-default-view';
import { WorkflowResource, parseWorkflowDefinition, getWorkflowInputs, getInputLabel, stringifyInputType } from 'models/workflow';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { ResourceOwnerWithName } from 'views-components/data-explorer/renderers';
import { formatDate } from "common/formatters";

export type CssRules = 'root' | 'tab' | 'inputTab' | 'graphTab' | 'graphTabWithChosenWorkflow' | 'descriptionTab' | 'inputsTable';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        height: '100%'
    },
    tab: {
        minWidth: '33%'
    },
    inputTab: {
        overflow: 'auto',
        maxHeight: '300px',
        marginTop: theme.spacing.unit
    },
    graphTab: {
        marginTop: theme.spacing.unit,
    },
    graphTabWithChosenWorkflow: {
        overflow: 'auto',
        height: '450px',
        marginTop: theme.spacing.unit,
    },
    descriptionTab: {
        overflow: 'auto',
        maxHeight: '300px',
        marginTop: theme.spacing.unit,
    },
    inputsTable: {
        tableLayout: 'fixed',
    },
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
                    <Tab className={classes.tab} label="Details" />
                </Tabs>
                {value === 0 && <CardContent className={classes.descriptionTab}>
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
                {value === 2 && <CardContent className={classes.descriptionTab}>
                    {workflow
                        ? <WorkflowDetailsAttributes workflow={workflow} />
                        : <DataTableDefaultView
                            icon={WorkflowIcon}
                            messages={['Please select a workflow to see its details.']} />
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
            return undefined;
        }

        renderInputsTable() {
            return <Table className={this.props.classes.inputsTable}>
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

export const WorkflowDetailsAttributes = ({ workflow }: WorkflowDetailsCardDataProps) => {
    return <Grid container>
        <Grid item xs={12} >
            <DetailsAttribute
                label={"Workflow UUID"}
                linkToUuid={workflow?.uuid} />
        </Grid>
        <Grid item xs={12} >
            <DetailsAttribute
                label='Owner' linkToUuid={workflow?.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceOwnerWithName uuid={uuid} />} />
        </Grid>
        <Grid item xs={12}>
            <DetailsAttribute label='Created at' value={formatDate(workflow?.createdAt)} />
        </Grid>
        <Grid item xs={12}>
            <DetailsAttribute label='Last modified' value={formatDate(workflow?.modifiedAt)} />
        </Grid>
        <Grid item xs={12} >
            <DetailsAttribute
                label='Last modified by user' linkToUuid={workflow?.modifiedByUserUuid}
                uuidEnhancer={(uuid: string) => <ResourceOwnerWithName uuid={uuid} />} />
        </Grid>
    </Grid >;
};
