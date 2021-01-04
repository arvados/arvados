// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogActions, Button, StyleRulesCallback, WithStyles, withStyles, CardHeader, Tab, Tabs } from '@material-ui/core';
import { withDialog } from "~/store/dialog/with-dialog";
import { COLLECTION_WEBDAV_S3_DIALOG_NAME, WebDavS3InfoDialogData } from '~/store/collections/collection-info-actions';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { compose } from 'redux';
import { DetailsAttribute } from "~/components/details-attribute/details-attribute";

type CssRules = 'details';

const styles: StyleRulesCallback<CssRules> = theme => ({
    details: {
        marginLeft: theme.spacing.unit * 3,
        marginRight: theme.spacing.unit * 3,
    }
});

interface TabPanelData {
    children: React.ReactElement<any>[];
    value: number;
    index: number;
}

function TabPanel(props: TabPanelData) {
    const { children, value, index } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`simple-tabpanel-${index}`}
            aria-labelledby={`simple-tab-${index}`}
        >
            {value === index && children}
        </div>
    );
}

export const WebDavS3InfoDialog = compose(
    withDialog(COLLECTION_WEBDAV_S3_DIALOG_NAME),
    withStyles(styles),
)(
    (props: WithDialogProps<WebDavS3InfoDialogData> & WithStyles<CssRules>) => {
        if (!props.data.downloadUrl) { return null; }

        const keepwebUrl = props.data.downloadUrl.replace(/\/\*(--[^.]+)?\./, "/");

        const winDav = new URL(props.data.downloadUrl.replace("*", props.data.uuid));

        const gnomeDav = new URL(keepwebUrl);
        gnomeDav.username = props.data.username;
        gnomeDav.pathname = `/c=${props.data.uuid}/`;
        gnomeDav.protocol = "davs:";

        const s3endpoint = new URL(keepwebUrl);

        const sp = props.data.token.split("/");
        let tokenUuid;
        let tokenSecret;
        if (sp.length === 3 && sp[0] === "v2" && props.data.homeCluster === props.data.localCluster) {
            tokenUuid = sp[1];
            tokenSecret = sp[2];
        } else {
            tokenUuid = props.data.token.replace(/\//g, "_");
            tokenSecret = tokenUuid;
        }

        return <Dialog
            open={props.open}
            maxWidth="md"
            onClose={props.closeDialog}
            style={{ alignSelf: 'stretch' }}>
            <CardHeader
                title={`WebDAV and S3`} />
            <div className={props.classes.details} >
                <Tabs value={props.data.activeTab} onChange={props.data.setActiveTab}>
                    <Tab key="windows" label="Add a Network Location in Windows" />
                    <Tab key="gnome" label="Connect to Server in GNOME" />
                    <Tab key="s3" label="Using an S3 client" />
                </Tabs>

                <TabPanel index={0} value={props.data.activeTab}>
                    <ol>
                        <li>Open File Explorer</li>
                        <li>Click on "This PC", then go to Computer &rarr; Add a Network Location</li>
                        <li>Click Next, then choose "Add a custom network location", then click Next</li>
                    </ol>

                    <DetailsAttribute
                        label='Internet address'
                        value={winDav.toString()}
                        copyValue={winDav.toString()} />

                    <DetailsAttribute
                        label='Username'
                        value={props.data.username}
                        copyValue={props.data.username} />

                    <DetailsAttribute
                        label='Password'
                        value={props.data.token}
                        copyValue={props.data.token} />
                </TabPanel>

                <TabPanel index={1} value={props.data.activeTab}>
                    <ol>
                        <li>Open Files</li>
                        <li>Select +Other Locations</li>
                        <li>Connect to Server &rarr; Enter server address</li>
                    </ol>

                    <DetailsAttribute
                        label='Server address'
                        value={gnomeDav.toString()}
                        copyValue={gnomeDav.toString()} />

                    <DetailsAttribute
                        label='Password'
                        value={props.data.token}
                        copyValue={props.data.token} />
                </TabPanel>

                <TabPanel index={2} value={props.data.activeTab}>
                    <DetailsAttribute
                        label='Endpoint'
                        value={s3endpoint.host}
                        copyValue={s3endpoint.host} />

                    <DetailsAttribute
                        label='Bucket'
                        value={props.data.uuid}
                        copyValue={props.data.uuid} />

                    <DetailsAttribute
                        label='Access Key'
                        value={tokenUuid}
                        copyValue={tokenUuid} />

                    <DetailsAttribute
                        label='Secret Key'
                        value={tokenSecret}
                        copyValue={tokenSecret} />

                </TabPanel>

            </div>
            <DialogActions>
                <Button
                    variant='text'
                    color='primary'
                    onClick={props.closeDialog}>
                    Close
		</Button>
            </DialogActions>

        </Dialog >;
    }
);
