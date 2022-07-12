// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardHeader,
    IconButton,
    CardContent,
    Tooltip,
    Typography,
    Tabs,
    Tab,
    Table,
    TableHead,
    TableBody,
    TableRow,
    TableCell,
    Paper,
    Link,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon, ProcessIcon } from 'components/icon/icon';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import {
  BooleanCommandInputParameter,
  CommandInputParameter,
  CWLType,
  Directory,
  DirectoryArrayCommandInputParameter,
  DirectoryCommandInputParameter,
  EnumCommandInputParameter,
  FileArrayCommandInputParameter,
  FileCommandInputParameter,
  FloatArrayCommandInputParameter,
  FloatCommandInputParameter,
  IntArrayCommandInputParameter,
  IntCommandInputParameter,
  isArrayOfType,
  isPrimitiveOfType,
  StringArrayCommandInputParameter,
  StringCommandInputParameter,
} from "models/workflow";
import { CommandOutputParameter } from 'cwlts/mappings/v1.0/CommandOutputParameter';
import { File } from 'models/workflow';
import { getInlineFileUrl } from 'views-components/context-menu/actions/helpers';
import { AuthState } from 'store/auth/auth-reducer';
import mime from 'mime';

type CssRules = 'card' | 'content' | 'title' | 'header' | 'avatar' | 'iconHeader' | 'tableWrapper' | 'tableRoot' | 'paramValue' | 'keepLink' | 'imagePreview';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    content: {
        padding: theme.spacing.unit * 1.0,
        paddingTop: theme.spacing.unit * 0.5,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
    },
    tableWrapper: {
        overflow: 'auto',
    },
    tableRoot: {
        width: '100%',
    },
    paramValue: {
        display: 'flex',
        alignItems: 'center',
    },
    keepLink: {
        cursor: 'pointer',
    },
    imagePreview: {
        maxHeight: '15em',
    },
});

export interface ProcessIOCardDataProps {
    label: string;
    params: ProcessIOParameter[];
    raw?: any;
}

type ProcessIOCardProps = ProcessIOCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessIOCard = withStyles(styles)(
    ({ classes, label, params, raw, doHidePanel, panelName }: ProcessIOCardProps) => {
        const [tabState, setTabState] = useState(0);
        const handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            setTabState(value);
        }

        return <Card className={classes.card}>
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                    avatar: classes.avatar,
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                title={
                    <Typography noWrap variant='h6' color='inherit'>
                        {label}
                    </Typography>
                }
                action={
                    <div>
                        { doHidePanel &&
                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doHidePanel}><CloseIcon /></IconButton>
                        </Tooltip> }
                    </div>
                } />
            <CardContent className={classes.content}>
                <Tabs value={tabState} onChange={handleChange} variant="fullWidth">
                    <Tab label="Preview" />
                    <Tab label="Raw" />
                </Tabs>
                {tabState === 0 && <div className={classes.tableWrapper}>
                    <ProcessIOPreview data={params} />
                    </div>}
                {tabState === 1 && <div className={classes.tableWrapper}>
                    <ProcessIORaw data={raw || params} />
                    </div>}
            </CardContent>
        </Card>;
    }
);

export type ProcessIOValue = {
    display: string;
    nav?: string;
    imageUrl?: string;
}

export type ProcessIOParameter = {
    id: string;
    doc: string;
    value: ProcessIOValue[];
}

interface ProcessIOPreviewDataProps {
    data: ProcessIOParameter[];
}

type ProcessIOPreviewProps = ProcessIOPreviewDataProps & WithStyles<CssRules>;

const ProcessIOPreview = withStyles(styles)(
    ({ classes, data }: ProcessIOPreviewProps) =>
        <Table className={classes.tableRoot} aria-label="simple table">
            <TableHead>
                <TableRow>
                    <TableCell>Label</TableCell>
                    <TableCell>Description</TableCell>
                    <TableCell>Value</TableCell>
                </TableRow>
            </TableHead>
            <TableBody>
                {data.map((param: ProcessIOParameter) => {
                    return <TableRow key={param.id}>
                        <TableCell component="th" scope="row">
                            {param.id}
                        </TableCell>
                        <TableCell>{param.doc}</TableCell>
                        <TableCell>{param.value.map(val => (
                            <Typography className={classes.paramValue}>
                                {val.imageUrl ? <img className={classes.imagePreview} src={val.imageUrl} alt="Inline Preview" /> : ""}
                                {val.nav ? <Link className={classes.keepLink} onClick={() => handleClick(val.nav)}>{val.display}</Link> : val.display}
                            </Typography>
                        ))}</TableCell>
                    </TableRow>;
                })}
            </TableBody>
        </Table>
);

const handleClick = (url) => {
    window.open(url, '_blank');
}

const ProcessIORaw = withStyles(styles)(
    ({ data }: ProcessIOPreviewProps) =>
        <Paper elevation={0}>
            <pre>
                {JSON.stringify(data, null, 2)}
            </pre>
        </Paper>
);

// secondaryFiles File[] is not part of CommandOutputParameter so we pass in an extra param
export const getInputDisplayValue = (auth: AuthState, input: CommandInputParameter | CommandOutputParameter, pdh?: string, secondaryFiles: File[] = []): ProcessIOValue[] => {
    switch (true) {
        case isPrimitiveOfType(input, CWLType.BOOLEAN):
            return [{display: String((input as BooleanCommandInputParameter).value)}];

        case isPrimitiveOfType(input, CWLType.INT):
        case isPrimitiveOfType(input, CWLType.LONG):
            return [{display: String((input as IntCommandInputParameter).value)}];

        case isPrimitiveOfType(input, CWLType.FLOAT):
        case isPrimitiveOfType(input, CWLType.DOUBLE):
            return [{display: String((input as FloatCommandInputParameter).value)}];

        case isPrimitiveOfType(input, CWLType.STRING):
            return [{display: (input as StringCommandInputParameter).value || ""}];

        case isPrimitiveOfType(input, CWLType.FILE):
            const mainFile = (input as FileCommandInputParameter).value;
            const files = [
                ...(mainFile ? [mainFile] : []),
                ...secondaryFiles
            ];
            return files.map(file => fileToProcessIOValue(file, auth, pdh));

        case isPrimitiveOfType(input, CWLType.DIRECTORY):
            const directory = (input as DirectoryCommandInputParameter).value;
            return directory ? [directoryToProcessIOValue(directory, auth, pdh)] : [];

        case typeof input.type === 'object' &&
            !(input.type instanceof Array) &&
            input.type.type === 'enum':
            return [{ display: (input as EnumCommandInputParameter).value || '' }];

        case isArrayOfType(input, CWLType.STRING):
            return [{ display: ((input as StringArrayCommandInputParameter).value || []).join(', ') }];

        case isArrayOfType(input, CWLType.INT):
        case isArrayOfType(input, CWLType.LONG):
            return [{ display: ((input as IntArrayCommandInputParameter).value || []).join(', ') }];

        case isArrayOfType(input, CWLType.FLOAT):
        case isArrayOfType(input, CWLType.DOUBLE):
            return [{ display: ((input as FloatArrayCommandInputParameter).value || []).join(', ') }];

        case isArrayOfType(input, CWLType.FILE):
            return ((input as FileArrayCommandInputParameter).value || [])
                .map(file => fileToProcessIOValue(file, auth, pdh));

        case isArrayOfType(input, CWLType.DIRECTORY):
            const directories = (input as DirectoryArrayCommandInputParameter).value || [];
            return directories.map(directory => directoryToProcessIOValue(directory, auth, pdh));

        default:
            return [];
    }
};

const getKeepUrl = (file: File | Directory, pdh?: string): string => {
    const isKeepUrl = file.location?.startsWith('keep:') || false;
    const keepUrl = isKeepUrl ? file.location : pdh ? `keep:${pdh}/${file.location}` : file.location;
    return keepUrl || '';
};

const getNavUrl = (auth: AuthState, file: File | Directory, pdh?: string): string => {
    let keepUrl = getKeepUrl(file, pdh).replace('keep:', '');
    return (getInlineFileUrl(`${auth.config.keepWebServiceUrl}/c=${keepUrl}?api_token=${auth.apiToken}`, auth.config.keepWebServiceUrl, auth.config.keepWebInlineServiceUrl));
};

const getImageUrl = (auth: AuthState, file: File, pdh?: string): string => {
    const keepUrl = getKeepUrl(file, pdh).replace('keep:', '');
    return getInlineFileUrl(`${auth.config.keepWebServiceUrl}/c=${keepUrl}?api_token=${auth.apiToken}`, auth.config.keepWebServiceUrl, auth.config.keepWebInlineServiceUrl);
};

const isFileImage = (basename?: string): boolean => {
    return basename ? (mime.getType(basename) || "").startsWith('image/') : false;
};

const normalizeDirectoryLocation = (directory: Directory): Directory => ({
    ...directory,
    location: (directory.location || '').endsWith('/') ? directory.location : directory.location + '/',
});

const directoryToProcessIOValue = (directory: Directory, auth: AuthState, pdh?: string): ProcessIOValue => {
    const normalizedDirectory = normalizeDirectoryLocation(directory);
    return {
        display: getKeepUrl(normalizedDirectory, pdh),
        nav: getNavUrl(auth, normalizedDirectory, pdh),
    };
};

const fileToProcessIOValue = (file: File, auth: AuthState, pdh?: string): ProcessIOValue => ({
    display: getKeepUrl(file, pdh),
    nav: getNavUrl(auth, file, pdh),
    imageUrl: isFileImage(file.basename) ? getImageUrl(auth, file, pdh) : undefined,
});
