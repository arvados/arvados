// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogActions, Button, StyleRulesCallback, WithStyles, withStyles, DialogTitle, DialogContent, Tabs, Tab, DialogContentText } from '@material-ui/core';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { withDialog } from "~/store/dialog/with-dialog";
import { compose } from 'redux';
import { ADVANCED_TAB_DIALOG } from "~/store/advanced-tab/advanced-tab";
import { DefaultCodeSnippet } from "~/components/default-code-snippet/default-code-snippet";

type CssRules = 'content' | 'codeSnippet' | 'secondContentText';

const styles: StyleRulesCallback<CssRules> = theme => ({
    content: {
        paddingTop: theme.spacing.unit * 3
    },
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: '1px solid',
        borderColor: theme.palette.grey["400"]
    },
    secondContentText: {
        paddingTop: theme.spacing.unit * 2
    }
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
                pythonHeader,
                pythonExample,
                CLIGetHeader,
                CLIGetExample,
                CLIUpdateHeader,
                CLIUpdateExample,
                curlHeader,
                curlExample
            } = this.props.data;
            return <Dialog
                open={open}
                maxWidth="md"
                onClose={closeDialog}
                onExit={() => this.setState({ value: 0 })} >
                <DialogTitle>Advanced</DialogTitle>
                <Tabs value={value} onChange={this.handleChange}>
                    <Tab label="API RESPONSE" />
                    <Tab label="METADATA" />
                    <Tab label="PYTHON EXAMPLE" />
                    <Tab label="CLI EXAMPLE" />
                    <Tab label="CURL EXAMPLE" />
                </Tabs>
                <DialogContent className={classes.content}>
                    {value === 0 && <div>
                        API CONTENT
                    </div>}
                    {value === 1 && <div>
                        METADATA CONTENT
                    </div>}
                    {value === 2 && <div>
                        <DialogContentText>{pythonHeader}</DialogContentText>
                        <DefaultCodeSnippet
                            className={classes.codeSnippet}
                            lines={[pythonExample]} />
                    </div>}
                    {value === 3 && <div>
                        <DialogContentText>{CLIGetHeader}</DialogContentText>
                        <DefaultCodeSnippet
                            className={classes.codeSnippet}
                            lines={[CLIGetExample]} />
                        <DialogContentText className={classes.secondContentText}>{CLIUpdateHeader}</DialogContentText>
                        <DefaultCodeSnippet
                            className={classes.codeSnippet}
                            lines={[CLIUpdateExample]} />
                    </div>}
                    {value === 4 && <div>
                        <DialogContentText>{curlHeader}</DialogContentText>
                        <DefaultCodeSnippet
                            className={classes.codeSnippet}
                            lines={[curlExample]} />
                    </div>}
                </DialogContent>
                <DialogActions>
                    <Button variant='flat' color='primary' onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>;
        }
    }
);