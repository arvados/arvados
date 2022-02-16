// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dialog, DialogActions, Button, StyleRulesCallback, WithStyles, withStyles, DialogTitle, DialogContent, Tabs, Tab, DialogContentText } from '@material-ui/core';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { withDialog } from "store/dialog/with-dialog";
import { compose } from 'redux';
import { ADVANCED_TAB_DIALOG } from "store/advanced-tab/advanced-tab";
import { DefaultCodeSnippet } from "components/default-code-snippet/default-code-snippet";
import { MetadataTab } from 'views-components/advanced-tab-dialog/metadataTab';

type CssRules = 'content' | 'codeSnippet' | 'spacing';

const styles: StyleRulesCallback<CssRules> = theme => ({
    content: {
        paddingTop: theme.spacing.unit * 3,
        minHeight: '400px',
        minWidth: '1232px'
    },
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: '1px solid',
        borderColor: theme.palette.grey["400"],
        maxHeight: '400px'
    },
    spacing: {
        paddingBottom: theme.spacing.unit * 2
    },
});

export const AdvancedTabDialog = compose(
    withDialog(ADVANCED_TAB_DIALOG),
    withStyles(styles),
)(
    class extends React.Component<WithDialogProps<any> & WithStyles<CssRules>>{
        state = {
            value: 0,
        };

        componentDidMount() {
            this.setState({ value: 0 });
        }

        handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            this.setState({ value });
        }
        render() {
            const { classes, open, closeDialog } = this.props;
            const { value } = this.state;
            const {
                apiResponse,
                metadata,
                pythonHeader,
                pythonExample,
                cliGetHeader,
                cliGetExample,
                cliUpdateHeader,
                cliUpdateExample,
                curlHeader,
                curlExample,
                uuid,
            } = this.props.data;
            return <Dialog
                open={open}
                maxWidth="lg"
                onClose={closeDialog}
                onExit={() => this.setState({ value: 0 })} >
                <DialogTitle>Advanced</DialogTitle>
                <Tabs value={value} onChange={this.handleChange} fullWidth>
                    <Tab label="API RESPONSE" />
                    <Tab label="METADATA" />
                    <Tab label="PYTHON EXAMPLE" />
                    <Tab label="CLI EXAMPLE" />
                    <Tab label="CURL EXAMPLE" />
                </Tabs>
                <DialogContent className={classes.content}>
                    {value === 0 && <div>{dialogContentExample(apiResponse, classes)}</div>}
                    {value === 1 && <div>
                        {metadata !== '' && metadata.items.length > 0 ?
                            <MetadataTab items={metadata.items} uuid={uuid} />
                            : dialogContentHeader('(No metadata links found)')}
                    </div>}
                    {value === 2 && dialogContent(pythonHeader, pythonExample, classes)}
                    {value === 3 && <div>
                        {dialogContent(cliGetHeader, cliGetExample, classes)}
                        {dialogContent(cliUpdateHeader, cliUpdateExample, classes)}
                    </div>}
                    {value === 4 && dialogContent(curlHeader, curlExample, classes)}
                </DialogContent>
                <DialogActions>
                    <Button data-cy="close-advanced-dialog" variant='text' color='primary' onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>;
        }
    }
);

const dialogContent = (header: string, example: string, classes: any) =>
    <div className={classes.spacing}>
        {dialogContentHeader(header)}
        {dialogContentExample(example, classes)}
    </div>;

const dialogContentHeader = (header: string) =>
    <DialogContentText>
        {header}
    </DialogContentText>;

const dialogContentExample = (example: string, classes: any) =>
    <DefaultCodeSnippet
        apiResponse
        className={classes.codeSnippet}
        lines={[example]} />;