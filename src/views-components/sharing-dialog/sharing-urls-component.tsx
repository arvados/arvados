// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Grid,
    IconButton,
    Link,
    StyleRulesCallback,
    Tooltip,
    Typography,
    WithStyles,
    withStyles
} from '@material-ui/core';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { CopyIcon, RemoveIcon } from 'components/icon/icon';
import CopyToClipboard from 'react-copy-to-clipboard';
import { ArvadosTheme } from 'common/custom-theme';
import moment from 'moment';

type CssRules = 'sharingUrlText'
    | 'sharingUrlButton'
    | 'sharingUrlList'
    | 'sharingUrlRow';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    sharingUrlText: {
        fontSize: '1rem',
    },
    sharingUrlButton: {
        color: theme.palette.grey["500"],
        cursor: 'pointer',
        '& svg': {
            fontSize: '1rem'
        },
        verticalAlign: 'middle',
    },
    sharingUrlList: {
        marginTop: '1rem',
    },
    sharingUrlRow: {
        borderBottom: `1px solid ${theme.palette.grey["300"]}`,
    },
});

export interface SharingURLsComponentDataProps {
    collectionUuid: string;
    sharingTokens: ApiClientAuthorization[];
    sharingURLsPrefix: string;
}

export interface SharingURLsComponentActionProps {
    onDeleteSharingToken: (uuid: string) => void;
    onCopy: (message: string) => void;
}

export type SharingURLsComponentProps = SharingURLsComponentDataProps & SharingURLsComponentActionProps;

export const SharingURLsComponent = withStyles(styles)((props: SharingURLsComponentProps & WithStyles<CssRules>) => <Grid container direction='column' spacing={24} className={props.classes.sharingUrlList}>
    { props.sharingTokens.length > 0
    ? props.sharingTokens
    .sort((a, b) => (new Date(a.expiresAt).getTime() - new Date(b.expiresAt).getTime()))
    .map(token => {
        const url = props.sharingURLsPrefix.includes('*')
        ? `${props.sharingURLsPrefix.replace('*', props.collectionUuid)}/t=${token.apiToken}/_/`
        : `${props.sharingURLsPrefix}/c=${props.collectionUuid}/t=${token.apiToken}/_/`;
        const expDate = new Date(token.expiresAt);
        const urlLabel = `Token ${token.apiToken.slice(0, 8)}... expiring at: ${expDate.toLocaleString()} (${moment(expDate).fromNow()})`;

        return <Grid container alignItems='center' key={token.uuid}  className={props.classes.sharingUrlRow}>
            <Grid item>
            <Link className={props.classes.sharingUrlText} href={url} target='_blank'>
                {urlLabel}
            </Link>
            </Grid>
            <Grid item xs />
            <Grid item>
            <span className={props.classes.sharingUrlButton}><Tooltip title='Copy to clipboard'>
                <CopyToClipboard text={url} onCopy={() => props.onCopy('Sharing URL copied')}>
                    <CopyIcon />
                </CopyToClipboard>
            </Tooltip></span>
            <span className={props.classes.sharingUrlButton}><Tooltip title='Remove'>
                <IconButton onClick={() => props.onDeleteSharingToken(token.uuid)}>
                    <RemoveIcon />
                </IconButton>
            </Tooltip></span>
            </Grid>
        </Grid>
    })
    : <Grid item><Typography>No sharing URLs</Typography></Grid> }
</Grid>);
