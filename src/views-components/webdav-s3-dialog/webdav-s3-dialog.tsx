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
import { DownloadIcon } from "~/components/icon/icon";

export type CssRules = 'details' | 'downloadButton';

const styles: StyleRulesCallback<CssRules> = theme => ({
    details: {
        marginLeft: theme.spacing.unit * 3,
        marginRight: theme.spacing.unit * 3,
    },
    downloadButton: {
        marginTop: theme.spacing.unit * 2,
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

const mountainduckTemplate = ({
    uuid, 
    username,
    collectionsUrl,
}: any) => `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
   <dict>
      <key>Protocol</key>
      <string>davs</string>
      <key>Provider</key>
      <string>iterate GmbH</string>
      <key>UUID</key>
      <string>${uuid}</string>
      <key>Hostname</key>
      <string>${collectionsUrl.replace('https://', ``).replace('*', uuid)}</string>
      <key>Port</key>
      <string>443</string>
      <key>Username</key>
      <string>${username}</string>
      <key>Labels</key>
      <array>
      </array>
   </dict>
</plist>
`.split(/\r?\n/).join('\n');

const downloadMountainduckFileHandler = (filename: string, text: string) => {
    const element = document.createElement('a');
    element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(text));
    element.setAttribute('download', filename);
  
    element.style.display = 'none';
    document.body.appendChild(element);
  
    element.click();
  
    document.body.removeChild(element);
};

export const WebDavS3InfoDialog = compose(
    withDialog(COLLECTION_WEBDAV_S3_DIALOG_NAME),
    withStyles(styles),
)(
    (props: WithDialogProps<WebDavS3InfoDialogData> & WithStyles<CssRules>) => {
        if (!props.data.downloadUrl) { return null; }

        let winDav;
        let cyberDav;

        if (props.data.collectionsUrl.indexOf("*") > -1) {
            const withuuid = props.data.collectionsUrl.replace("*", props.data.uuid);
            winDav = new URL(withuuid);
            cyberDav = new URL(withuuid);
        } else {
            winDav = new URL(props.data.downloadUrl);
            cyberDav = new URL(props.data.downloadUrl);
            winDav.pathname = `/by_id/${props.data.uuid}`;
            cyberDav.pathname = `/by_id/${props.data.uuid}`;
        }

        cyberDav.username = props.data.username;
        const cyberDavStr = "dav" + cyberDav.toString().slice(4);

        const s3endpoint = new URL(props.data.collectionsUrl.replace(/\/\*(--[^.]+)?\./, "/"));

        const sp = props.data.token.split("/");
        let tokenUuid;
        let tokenSecret;
        if (sp.length === 3 && sp[0] === "v2" && sp[1].slice(0, 5) === props.data.localCluster) {
            tokenUuid = sp[1];
            tokenSecret = sp[2];
        } else {
            tokenUuid = props.data.token.replace(/\//g, "_");
            tokenSecret = tokenUuid;
        }

        const supportsWebdav = (props.data.uuid.indexOf("-4zz18-") === 5);

        let activeTab = props.data.activeTab;
        if (!supportsWebdav) {
            activeTab = 2;
        }

        return <Dialog
            open={props.open}
            maxWidth="md"
            onClose={props.closeDialog}
            style={{ alignSelf: 'stretch' }}>
            <CardHeader
                title={`Open as Network Folder or S3 Bucket`} />
            <div className={props.classes.details} >
                <Tabs value={activeTab} onChange={props.data.setActiveTab}>
                    {supportsWebdav && <Tab value={0} key="cyberduck" label="Cyberduck/Mountain Duck or Gnome Files" />}
                    {supportsWebdav && <Tab value={1} key="windows" label="Windows or MacOS" />}
                    <Tab value={2} key="s3" label="S3 bucket" />
                </Tabs>

                <TabPanel index={1} value={activeTab}>
                    <h2>Settings</h2>

                    <DetailsAttribute
                        label='Internet address'
                        value={<a href={winDav.toString()} target="_blank">{winDav.toString()}</a>}
                        copyValue={winDav.toString()} />

                    <DetailsAttribute
                        label='Username'
                        value={props.data.username}
                        copyValue={props.data.username} />

                    <DetailsAttribute
                        label='Password'
                        value={props.data.token}
                        copyValue={props.data.token} />

                    <h3>Windows</h3>
                    <ol>
                        <li>Open File Explorer</li>
                        <li>Click on "This PC", then go to Computer &rarr; Add a Network Location</li>
                        <li>Click Next, then choose "Add a custom network location", then click Next</li>
                    </ol>

                    <h3>MacOS</h3>
                    <ol>
                        <li>Open Finder</li>
                        <li>Click Go &rarr; Connect to server</li>
                    </ol>
                </TabPanel>

                <TabPanel index={0} value={activeTab}>
                    <DetailsAttribute
                        label='Server'
                        value={<a href={cyberDavStr} target="_blank">{cyberDavStr}</a>}
                        copyValue={cyberDavStr} />

                    <DetailsAttribute
                        label='Username'
                        value={props.data.username}
                        copyValue={props.data.username} />

                    <DetailsAttribute
                        label='Password'
                        value={props.data.token}
                        copyValue={props.data.token} />

                    <Button
                        data-cy='download-button'
                        className={props.classes.downloadButton}
                        onClick={() => downloadMountainduckFileHandler(`${props.data.collectionName || props.data.uuid}.duck`, mountainduckTemplate(props.data))}
                        variant='contained'
                        color='primary'
                        size='small'>
                        <DownloadIcon />
                        Download config
                    </Button>

                    <h3>Gnome</h3>
                    <ol>
                        <li>Open Files</li>
                        <li>Select +Other Locations</li>
                        <li>Connect to Server &rarr; Enter server address</li>
                    </ol>

                </TabPanel>

                <TabPanel index={2} value={activeTab}>
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
