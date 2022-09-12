// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement, useState } from 'react';
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
    Grid,
    Chip,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon, InfoIcon, ProcessIcon, InputIcon, OutputIcon } from 'components/icon/icon';
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
import { DefaultView } from 'components/default-view/default-view';
import { getNavUrl } from 'routes/routes';
import { Link as RouterLink } from 'react-router-dom';
import { Link as MuiLink } from '@material-ui/core';
import { InputCollectionMount } from 'store/processes/processes-actions';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { ProcessOutputCollectionFiles } from './process-output-collection-files';

type CssRules = 'card' | 'content' | 'title' | 'header' | 'avatar' | 'iconHeader' | 'tableWrapper' | 'tableRoot' | 'paramValue' | 'keepLink' | 'imagePreview' | 'valArray' | 'emptyValue';

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
        alignItems: 'flex-start',
        flexDirection: 'column',
    },
    keepLink: {
        color: theme.palette.primary.main,
        textDecoration: 'none',
        overflowWrap: 'break-word',
        cursor: 'pointer',
    },
    imagePreview: {
        maxHeight: '15em',
        maxWidth: '15em',
        marginBottom: theme.spacing.unit,
    },
    valArray: {
        display: 'flex',
        gap: '10px',
        flexWrap: 'wrap',
        '& span': {
            display: 'inline',
        }
    },
    emptyValue: {
        color: theme.customs.colors.grey500,
    },
});

export enum ProcessIOCardType {
    INPUT = 'Inputs',
    OUTPUT = 'Outputs',
}
export interface ProcessIOCardDataProps {
    label: ProcessIOCardType;
    params: ProcessIOParameter[];
    raw?: any;
    mounts?: InputCollectionMount[];
    outputUuid?: string;
}

type ProcessIOCardProps = ProcessIOCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessIOCard = withStyles(styles)(
    ({ classes, label, params, raw, mounts, outputUuid, doHidePanel, panelName }: ProcessIOCardProps) => {
        const [tabState, setTabState] = useState(0);
        const handleChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            setTabState(value);
        }
        const PanelIcon = label == ProcessIOCardType.INPUT ? InputIcon : OutputIcon;

        return <Card className={classes.card} data-cy="process-io-card">
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                    avatar: classes.avatar,
                }}
                avatar={<PanelIcon className={classes.iconHeader} />}
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
                <div>
                    <Tabs value={tabState} onChange={handleChange} variant="fullWidth">
                        <Tab label="Preview" />
                        <Tab label="Raw" />
                        {label === ProcessIOCardType.INPUT && <Tab label="Input Mounts" />}
                        {label === ProcessIOCardType.OUTPUT && <Tab label="Output Collection" />}
                    </Tabs>
                    {tabState === 0 && <div className={classes.tableWrapper}>
                        {params.length ?
                            <ProcessIOPreview data={params} /> :
                            <Grid container item alignItems='center' justify='center'>
                                <DefaultView messages={["No parameters found"]} icon={InfoIcon} />
                            </Grid>}
                        </div>}
                    {tabState === 1 && <div className={classes.tableWrapper}>
                        {params.length ?
                            <ProcessIORaw data={raw || params} /> :
                            <Grid container item alignItems='center' justify='center'>
                                <DefaultView messages={["No parameters found"]} icon={InfoIcon} />
                            </Grid>}
                        </div>}
                    {tabState === 2 && <div className={classes.tableWrapper}>
                        {label === ProcessIOCardType.INPUT && <ProcessInputMounts mounts={mounts || []} />}
                        {label === ProcessIOCardType.OUTPUT && <ProcessOutputCollectionFiles isWritable={false} currentItemUuid={outputUuid} />}
                        </div>}
                </div>
            </CardContent>
        </Card>;
    }
);

export type ProcessIOValue = {
    display: ReactElement<any, any>;
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
        <Table className={classes.tableRoot} aria-label="Process IO Preview">
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
                                <span className={classes.valArray}>
                                    {val.display}
                                </span>
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

interface ProcessInputMountsDataProps {
    mounts: InputCollectionMount[];
}

type ProcessInputMountsProps = ProcessInputMountsDataProps & WithStyles<CssRules>;

const ProcessInputMounts = withStyles(styles)(connect((state: RootState) => ({
    auth: state.auth,
}))(({ mounts, classes, auth }: ProcessInputMountsProps & { auth: AuthState }) => (
    <Table className={classes.tableRoot} aria-label="Process Input Mounts">
        <TableHead>
            <TableRow>
                <TableCell>Path</TableCell>
                <TableCell>Portable Data Hash</TableCell>
            </TableRow>
        </TableHead>
        <TableBody>
            {mounts.map(mount => (
                <TableRow key={mount.path}>
                    <TableCell><pre>{mount.path}</pre></TableCell>
                    <TableCell>
                        <RouterLink to={getNavUrl(mount.pdh, auth)} className={classes.keepLink}>{mount.pdh}</RouterLink>
                    </TableCell>
                </TableRow>
            ))}
        </TableBody>
    </Table>
)));

type FileWithSecondaryFiles = {
    secondaryFiles: File[];
}

export const getIOParamDisplayValue = (auth: AuthState, input: CommandInputParameter | CommandOutputParameter, pdh?: string): ProcessIOValue[] => {
    switch (true) {
        case isPrimitiveOfType(input, CWLType.BOOLEAN):
            const boolValue = (input as BooleanCommandInputParameter).value;

            return boolValue !== undefined &&
                    !(Array.isArray(boolValue) && boolValue.length === 0) ?
                [{display: <pre>{String(boolValue)}</pre> }] :
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.INT):
        case isPrimitiveOfType(input, CWLType.LONG):
            const intValue = (input as IntCommandInputParameter).value;

            return intValue !== undefined &&
                    // Missing values are empty array
                    !(Array.isArray(intValue) && intValue.length === 0) ?
                [{display: <pre>{String(intValue)}</pre> }]
                : [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.FLOAT):
        case isPrimitiveOfType(input, CWLType.DOUBLE):
            const floatValue = (input as FloatCommandInputParameter).value;

            return floatValue !== undefined &&
                    !(Array.isArray(floatValue) && floatValue.length === 0) ?
                [{display: <pre>{String(floatValue)}</pre> }]:
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.STRING):
            const stringValue = (input as StringCommandInputParameter).value || undefined;

            return stringValue !== undefined &&
                    !(Array.isArray(stringValue) && stringValue.length === 0) ?
                [{display: <pre>{stringValue}</pre> }] :
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.FILE):
            const mainFile = (input as FileCommandInputParameter).value;
            // secondaryFiles: File[] is not part of CommandOutputParameter so we cast to access secondaryFiles
            const secondaryFiles = ((mainFile as unknown) as FileWithSecondaryFiles)?.secondaryFiles || [];
            const files = [
                ...(mainFile && !(Array.isArray(mainFile) && mainFile.length === 0) ? [mainFile] : []),
                ...secondaryFiles
            ];

            return files.length ?
                files.map(file => fileToProcessIOValue(file, auth, pdh)) :
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.DIRECTORY):
            const directory = (input as DirectoryCommandInputParameter).value;

            return directory !== undefined &&
                    !(Array.isArray(directory) && directory.length === 0) ?
                [directoryToProcessIOValue(directory, auth, pdh)] :
                [{display: <EmptyValue />}];

        case typeof input.type === 'object' &&
            !(input.type instanceof Array) &&
            input.type.type === 'enum':
            const enumValue = (input as EnumCommandInputParameter).value;

            return enumValue !== undefined ?
                [{ display: <pre>{(input as EnumCommandInputParameter).value || ''}</pre> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.STRING):
            const strArray = (input as StringArrayCommandInputParameter).value || [];
            return strArray.length ?
                [{ display: <>{((input as StringArrayCommandInputParameter).value || []).map((val) => <Chip label={val} />)}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.INT):
        case isArrayOfType(input, CWLType.LONG):
            const intArray = (input as IntArrayCommandInputParameter).value || [];

            return intArray.length ?
                [{ display: <>{((input as IntArrayCommandInputParameter).value || []).map((val) => <Chip label={val} />)}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.FLOAT):
        case isArrayOfType(input, CWLType.DOUBLE):
            const floatArray = (input as FloatArrayCommandInputParameter).value || [];

            return floatArray.length ?
                [{ display: <>{floatArray.map((val) => <Chip label={val} />)}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.FILE):
            const fileArrayMainFile = ((input as FileArrayCommandInputParameter).value || []);
            const fileArraySecondaryFiles = fileArrayMainFile.map((file) => (
                ((file as unknown) as FileWithSecondaryFiles)?.secondaryFiles || []
            )).reduce((acc: File[], params: File[]) => (acc.concat(params)), []);

            const fileArrayFiles = [
                ...fileArrayMainFile,
                ...fileArraySecondaryFiles
            ];

            return fileArrayFiles.length ?
                fileArrayFiles.map(file => fileToProcessIOValue(file, auth, pdh)) :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.DIRECTORY):
            const directories = (input as DirectoryArrayCommandInputParameter).value || [];

            return directories.length ?
                directories.map(directory => directoryToProcessIOValue(directory, auth, pdh)) :
                [{display: <EmptyValue />}];

        default:
            return [];
    }
};

const getKeepUrl = (file: File | Directory, pdh?: string): string => {
    const isKeepUrl = file.location?.startsWith('keep:') || false;
    const keepUrl = isKeepUrl ? file.location : pdh ? `keep:${pdh}/${file.location}` : file.location;
    return keepUrl || '';
};

interface KeepUrlProps {
    auth: AuthState;
    res: File | Directory;
    pdh?: string;
}

const KeepUrlBase = withStyles(styles)(({auth, res, pdh, classes}: KeepUrlProps & WithStyles<CssRules>) => {
    const keepUrl = getKeepUrl(res, pdh);
    const pdhUrl = keepUrl ? keepUrl.split('/').slice(0, 1)[0] : '';
    // Passing a pdh always returns a relative wb2 collection url
    const pdhWbPath = getNavUrl(pdhUrl.replace('keep:', ''), auth);
    return pdhUrl && pdhWbPath ?
        <RouterLink to={pdhWbPath} className={classes.keepLink}>{pdhUrl}</RouterLink> :
        <></>;
});

const KeepUrlPath = withStyles(styles)(({auth, res, pdh, classes}: KeepUrlProps & WithStyles<CssRules>) => {
    const keepUrl = getKeepUrl(res, pdh);
    const keepUrlParts = keepUrl ? keepUrl.split('/') : [];
    const keepUrlPath = keepUrlParts.length > 1 ? keepUrlParts.slice(1).join('/') : '';

    const keepUrlPathNav = getKeepNavUrl(auth, res, pdh);
    return keepUrlPath && keepUrlPathNav ?
        <MuiLink className={classes.keepLink} onClick={() => handleClick(keepUrlPathNav)}>{keepUrlPath}</MuiLink> :
        <></>;
});

const getKeepNavUrl = (auth: AuthState, file: File | Directory, pdh?: string): string => {
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

const normalizeDirectoryLocation = (directory: Directory): Directory => {
    if (!directory.location) {
        return directory;
    }
    return {
        ...directory,
        location: (directory.location || '').endsWith('/') ? directory.location : directory.location + '/',
    };
};

const directoryToProcessIOValue = (directory: Directory, auth: AuthState, pdh?: string): ProcessIOValue => {
    const normalizedDirectory = normalizeDirectoryLocation(directory);
    return {
        display: <span>
            <KeepUrlBase auth={auth} res={normalizedDirectory} pdh={pdh}/>/<KeepUrlPath auth={auth} res={normalizedDirectory} pdh={pdh}/>
        </span>,
    };
};

const fileToProcessIOValue = (file: File, auth: AuthState, pdh?: string): ProcessIOValue => {
    return {
        display: <span>
            <KeepUrlBase auth={auth} res={file} pdh={pdh}/>/<KeepUrlPath auth={auth} res={file} pdh={pdh}/>
        </span>,
        imageUrl: isFileImage(file.basename) ? getImageUrl(auth, file, pdh) : undefined,
    }
};

const EmptyValue = withStyles(styles)(
    ({classes}: WithStyles<CssRules>) => <span className={classes.emptyValue}>No value</span>
);
