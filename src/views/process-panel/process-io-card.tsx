// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement, useState } from 'react';
import { Dispatch } from 'redux';
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
    CircularProgress,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import {
    CloseIcon,
    ImageIcon,
    InputIcon,
    ImageOffIcon,
    OutputIcon,
    MaximizeIcon,
    UnMaximizeIcon,
    InfoIcon
} from 'components/icon/icon';
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
import { Process } from 'store/processes/process';
import { navigateTo } from 'store/navigation/navigation-action';
import classNames from 'classnames';
import { DefaultCodeSnippet } from 'components/default-code-snippet/default-code-snippet';

type CssRules =
  | "card"
  | "content"
  | "title"
  | "header"
  | "avatar"
  | "iconHeader"
  | "tableWrapper"
  | "tableRoot"
  | "paramValue"
  | "keepLink"
  | "collectionLink"
  | "imagePreview"
  | "valArray"
  | "secondaryVal"
  | "secondaryRow"
  | "emptyValue"
  | "noBorderRow"
  | "symmetricTabs"
  | "imagePlaceholder"
  | "rowWithPreview"
  | "labelColumn";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: 0,
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
        height: `calc(100% - ${theme.spacing.unit * 7}px - ${theme.spacing.unit * 1.5}px)`,
        padding: theme.spacing.unit * 1.0,
        paddingTop: 0,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
    },
    tableWrapper: {
        height: `calc(100% - ${theme.spacing.unit * 6}px)`,
        overflow: 'auto',
    },
    tableRoot: {
        width: '100%',
        '& thead th': {
            verticalAlign: 'bottom',
            paddingBottom: '10px',
        },
        '& td, & th': {
            paddingRight: '25px',
        }
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
    collectionLink: {
        margin: '10px',
        '& a': {
            color: theme.palette.primary.main,
            textDecoration: 'none',
            overflowWrap: 'break-word',
            cursor: 'pointer',
        }
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
    secondaryVal: {
        paddingLeft: '20px',
    },
    secondaryRow: {
        height: '29px',
        verticalAlign: 'top',
        position: 'relative',
        top: '-9px',
    },
    emptyValue: {
        color: theme.customs.colors.grey500,
    },
    noBorderRow: {
        '& td': {
            borderBottom: 'none',
        }
    },
    symmetricTabs: {
        '& button': {
            flexBasis: '0',
        }
    },
    imagePlaceholder: {
        width: '60px',
        height: '60px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#cecece',
        borderRadius: '10px',
    },
    rowWithPreview: {
        verticalAlign: 'bottom',
    },
    labelColumn: {
        minWidth: '120px',
    },
});

export enum ProcessIOCardType {
    INPUT = 'Inputs',
    OUTPUT = 'Outputs',
}
export interface ProcessIOCardDataProps {
    process: Process;
    label: ProcessIOCardType;
    params: ProcessIOParameter[] | null;
    raw: any;
    mounts?: InputCollectionMount[];
    outputUuid?: string;
}

export interface ProcessIOCardActionProps {
    navigateTo: (uuid: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch): ProcessIOCardActionProps => ({
    navigateTo: (uuid) => dispatch<any>(navigateTo(uuid)),
});

type ProcessIOCardProps = ProcessIOCardDataProps & ProcessIOCardActionProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessIOCard = withStyles(styles)(connect(null, mapDispatchToProps)(
    ({ classes, label, params, raw, mounts, outputUuid, doHidePanel, doMaximizePanel, doUnMaximizePanel, panelMaximized, panelName, process, navigateTo }: ProcessIOCardProps) => {
        const [mainProcTabState, setMainProcTabState] = useState(0);
        const handleMainProcTabChange = (event: React.MouseEvent<HTMLElement>, value: number) => {
            setMainProcTabState(value);
        }

        const [showImagePreview, setShowImagePreview] = useState(false);

        const PanelIcon = label === ProcessIOCardType.INPUT ? InputIcon : OutputIcon;
        const mainProcess = !process.containerRequest.requestingContainerUuid;

        const loading = raw === null || raw === undefined || params === null;
        const hasRaw = !!(raw && Object.keys(raw).length > 0);
        const hasParams = !!(params && params.length > 0);

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
                        { mainProcess && <Tooltip title={"Toggle Image Preview"} disableFocusListener>
                            <IconButton data-cy="io-preview-image-toggle" onClick={() =>{setShowImagePreview(!showImagePreview)}}>{showImagePreview ? <ImageIcon /> : <ImageOffIcon />}</IconButton>
                        </Tooltip> }
                        { doUnMaximizePanel && panelMaximized &&
                        <Tooltip title={`Unmaximize ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doUnMaximizePanel}><UnMaximizeIcon /></IconButton>
                        </Tooltip> }
                        { doMaximizePanel && !panelMaximized &&
                        <Tooltip title={`Maximize ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doMaximizePanel}><MaximizeIcon /></IconButton>
                        </Tooltip> }
                        { doHidePanel &&
                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton disabled={panelMaximized} onClick={doHidePanel}><CloseIcon /></IconButton>
                        </Tooltip> }
                    </div>
                } />
            <CardContent className={classes.content}>
                {mainProcess ?
                    (<>
                        {/* raw is undefined until params are loaded */}
                        {loading && <Grid container item alignItems='center' justify='center'>
                            <CircularProgress />
                        </Grid>}
                        {/* Once loaded, either raw or params may still be empty
                          *   Raw when all params are empty
                          *   Params when raw is provided by containerRequest properties but workflow mount is absent for preview
                          */}
                        {(!loading && (hasRaw || hasParams)) &&
                            <>
                                <Tabs value={mainProcTabState} onChange={handleMainProcTabChange} variant="fullWidth" className={classes.symmetricTabs}>
                                    {/* params will be empty on processes without workflow definitions in mounts, so we only show raw */}
                                    {hasParams && <Tab label="Parameters" />}
                                    <Tab label="JSON" />
                                </Tabs>
                                {(mainProcTabState === 0 && params && hasParams) && <div className={classes.tableWrapper}>
                                        <ProcessIOPreview data={params} showImagePreview={showImagePreview} />
                                    </div>}
                                {(mainProcTabState === 1 || !hasParams) && <div className={classes.tableWrapper}>
                                        <ProcessIORaw data={raw} />
                                    </div>}
                            </>}
                        {!loading && !hasRaw && !hasParams && <Grid container item alignItems='center' justify='center'>
                            <DefaultView messages={["No parameters found"]} />
                        </Grid>}
                    </>) :
                    // Subprocess
                    (<>
                        {((mounts && mounts.length) || outputUuid) ?
                            <>
                                <Tabs value={0} variant="fullWidth" className={classes.symmetricTabs}>
                                    {label === ProcessIOCardType.INPUT && <Tab label="Collections" />}
                                    {label === ProcessIOCardType.OUTPUT && <Tab label="Collection" />}
                                </Tabs>
                                <div className={classes.tableWrapper}>
                                    {label === ProcessIOCardType.INPUT && <ProcessInputMounts mounts={mounts || []} />}
                                    {label === ProcessIOCardType.OUTPUT && <>
                                        {outputUuid && <Typography className={classes.collectionLink}>
                                            Output Collection: <MuiLink className={classes.keepLink} onClick={() => {navigateTo(outputUuid || "")}}>
                                            {outputUuid}
                                        </MuiLink></Typography>}
                                        <ProcessOutputCollectionFiles isWritable={false} currentItemUuid={outputUuid} />
                                    </>}
                                </div>
                            </> :
                            <Grid container item alignItems='center' justify='center'>
                                <DefaultView messages={["No collection(s) found"]} />
                            </Grid>
                        }
                    </>)
                }
            </CardContent>
        </Card>;
    }
));

export type ProcessIOValue = {
    display: ReactElement<any, any>;
    imageUrl?: string;
    collection?: ReactElement<any, any>;
    secondary?: boolean;
}

export type ProcessIOParameter = {
    id: string;
    label: string;
    value: ProcessIOValue[];
}

interface ProcessIOPreviewDataProps {
    data: ProcessIOParameter[];
    showImagePreview: boolean;
}

type ProcessIOPreviewProps = ProcessIOPreviewDataProps & WithStyles<CssRules>;

const ProcessIOPreview = withStyles(styles)(
    ({ classes, data, showImagePreview }: ProcessIOPreviewProps) => {
        const showLabel = data.some((param: ProcessIOParameter) => param.label);
        return <Table className={classes.tableRoot} aria-label="Process IO Preview">
            <TableHead>
                <TableRow>
                    <TableCell>Name</TableCell>
                    {showLabel && <TableCell className={classes.labelColumn}>Label</TableCell>}
                    <TableCell>Value</TableCell>
                    <TableCell>Collection</TableCell>
                </TableRow>
            </TableHead>
            <TableBody>
                {data.map((param: ProcessIOParameter) => {
                    const firstVal = param.value.length > 0 ? param.value[0] : undefined;
                    const rest = param.value.slice(1);
                    const mainRowClasses = {
                        [classes.noBorderRow]: (rest.length > 0),
                    };

                    return <>
                        <TableRow className={classNames(mainRowClasses)} data-cy="process-io-param">
                            <TableCell>
                                {param.id}
                            </TableCell>
                            {showLabel && <TableCell >{param.label}</TableCell>}
                            <TableCell>
                                {firstVal && <ProcessValuePreview value={firstVal} showImagePreview={showImagePreview} />}
                            </TableCell>
                            <TableCell className={firstVal?.imageUrl ? classes.rowWithPreview : undefined}>
                                <Typography className={classes.paramValue}>
                                    {firstVal?.collection}
                                </Typography>
                            </TableCell>
                        </TableRow>
                        {rest.map((val, i) => {
                            const rowClasses = {
                                [classes.noBorderRow]: (i < rest.length-1),
                                [classes.secondaryRow]: val.secondary,
                            };
                            return <TableRow className={classNames(rowClasses)}>
                                <TableCell />
                                {showLabel && <TableCell />}
                                <TableCell>
                                    <ProcessValuePreview value={val} showImagePreview={showImagePreview} />
                                </TableCell>
                                <TableCell className={firstVal?.imageUrl ? classes.rowWithPreview : undefined}>
                                    <Typography className={classes.paramValue}>
                                        {val.collection}
                                    </Typography>
                                </TableCell>
                            </TableRow>
                        })}
                    </>;
                })}
            </TableBody>
        </Table>;
});

interface ProcessValuePreviewProps {
    value: ProcessIOValue;
    showImagePreview: boolean;
}

const ProcessValuePreview = withStyles(styles)(
    ({value, showImagePreview, classes}: ProcessValuePreviewProps & WithStyles<CssRules>) =>
        <Typography className={classes.paramValue}>
            {value.imageUrl && showImagePreview ? <img className={classes.imagePreview} src={value.imageUrl} alt="Inline Preview" /> : ""}
            {value.imageUrl && !showImagePreview ? <ImagePlaceholder /> : ""}
            <span className={classNames(classes.valArray, value.secondary && classes.secondaryVal)}>
                {value.display}
            </span>
        </Typography>
)

interface ProcessIORawDataProps {
    data: ProcessIOParameter[];
}

const ProcessIORaw = withStyles(styles)(
    ({ data }: ProcessIORawDataProps) =>
        <Paper elevation={0}>
            <DefaultCodeSnippet lines={[JSON.stringify(data, null, 2)]} linked />
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
                [{display: renderPrimitiveValue(boolValue, false) }] :
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.INT):
        case isPrimitiveOfType(input, CWLType.LONG):
            const intValue = (input as IntCommandInputParameter).value;
            return intValue !== undefined &&
                    // Missing values are empty array
                    !(Array.isArray(intValue) && intValue.length === 0) ?
                [{display: renderPrimitiveValue(intValue, false) }]
                : [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.FLOAT):
        case isPrimitiveOfType(input, CWLType.DOUBLE):
            const floatValue = (input as FloatCommandInputParameter).value;
            return floatValue !== undefined &&
                    !(Array.isArray(floatValue) && floatValue.length === 0) ?
                [{display: renderPrimitiveValue(floatValue, false) }]:
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.STRING):
            const stringValue = (input as StringCommandInputParameter).value || undefined;
            return stringValue !== undefined &&
                    !(Array.isArray(stringValue) && stringValue.length === 0) ?
                [{display: renderPrimitiveValue(stringValue, false) }] :
                [{display: <EmptyValue />}];

        case isPrimitiveOfType(input, CWLType.FILE):
            const mainFile = (input as FileCommandInputParameter).value;
            // secondaryFiles: File[] is not part of CommandOutputParameter so we cast to access secondaryFiles
            const secondaryFiles = ((mainFile as unknown) as FileWithSecondaryFiles)?.secondaryFiles || [];
            const files = [
                ...(mainFile && !(Array.isArray(mainFile) && mainFile.length === 0) ? [mainFile] : []),
                ...secondaryFiles
            ];
            const mainFilePdhUrl = mainFile ? getResourcePdhUrl(mainFile, pdh) : "";
            return files.length ?
                files.map((file, i) => fileToProcessIOValue(file, (i > 0), auth, pdh, (i > 0 ? mainFilePdhUrl : ""))) :
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
            return enumValue !== undefined && enumValue ?
                [{ display: <pre>{enumValue}</pre> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.STRING):
            const strArray = (input as StringArrayCommandInputParameter).value || [];
            return strArray.length ?
                [{ display: <>{strArray.map((val) => renderPrimitiveValue(val, true))}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.INT):
        case isArrayOfType(input, CWLType.LONG):
            const intArray = (input as IntArrayCommandInputParameter).value || [];
            return intArray.length ?
                [{ display: <>{intArray.map((val) => renderPrimitiveValue(val, true))}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.FLOAT):
        case isArrayOfType(input, CWLType.DOUBLE):
            const floatArray = (input as FloatArrayCommandInputParameter).value || [];
            return floatArray.length ?
                [{ display: <>{floatArray.map((val) => renderPrimitiveValue(val, true))}</> }] :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.FILE):
            const fileArrayMainFiles = ((input as FileArrayCommandInputParameter).value || []);
            const firstMainFilePdh = (fileArrayMainFiles.length > 0 && fileArrayMainFiles[0]) ? getResourcePdhUrl(fileArrayMainFiles[0], pdh) : "";

            // Convert each main file into separate arrays of ProcessIOValue to preserve secondaryFile grouping
            const fileArrayValues = fileArrayMainFiles.map((mainFile: File, i): ProcessIOValue[] => {
                const secondaryFiles = ((mainFile as unknown) as FileWithSecondaryFiles)?.secondaryFiles || [];
                return [
                    // Pass firstMainFilePdh to secondary files and every main file besides the first to hide pdh if equal
                    ...(mainFile ? [fileToProcessIOValue(mainFile, false, auth, pdh, i > 0 ? firstMainFilePdh : "")] : []),
                    ...(secondaryFiles.map(file => fileToProcessIOValue(file, true, auth, pdh, firstMainFilePdh)))
                ];
            // Reduce each mainFile/secondaryFile group into single array preserving ordering
            }).reduce((acc: ProcessIOValue[], mainFile: ProcessIOValue[]) => (acc.concat(mainFile)), []);

            return fileArrayValues.length ?
                fileArrayValues :
                [{display: <EmptyValue />}];

        case isArrayOfType(input, CWLType.DIRECTORY):
            const directories = (input as DirectoryArrayCommandInputParameter).value || [];
            return directories.length ?
                directories.map(directory => directoryToProcessIOValue(directory, auth, pdh)) :
                [{display: <EmptyValue />}];

        default:
            return [{display: <UnsupportedValue />}];
    }
};

const renderPrimitiveValue = (value: any, asChip: boolean) => {
    const isObject = typeof value === 'object';
    if (!isObject) {
        return asChip ? <Chip label={String(value)} /> : <pre>{String(value)}</pre>;
    } else {
        return asChip ? <UnsupportedValueChip /> : <UnsupportedValue />;
    }
};

/*
 * @returns keep url without keep: prefix
 */
const getKeepUrl = (file: File | Directory, pdh?: string): string => {
    const isKeepUrl = file.location?.startsWith('keep:') || false;
    const keepUrl = isKeepUrl ?
                        file.location?.replace('keep:', '') :
                        pdh ? `${pdh}/${file.location}` : file.location;
    return keepUrl || '';
};

interface KeepUrlProps {
    auth: AuthState;
    res: File | Directory;
    pdh?: string;
}

const getResourcePdhUrl = (res: File | Directory, pdh?: string): string => {
    const keepUrl = getKeepUrl(res, pdh);
    return keepUrl ? keepUrl.split('/').slice(0, 1)[0] : '';
};

const KeepUrlBase = withStyles(styles)(({auth, res, pdh, classes}: KeepUrlProps & WithStyles<CssRules>) => {
    const pdhUrl = getResourcePdhUrl(res, pdh);
    // Passing a pdh always returns a relative wb2 collection url
    const pdhWbPath = getNavUrl(pdhUrl, auth);
    return pdhUrl && pdhWbPath ?
        <Tooltip title={"View collection in Workbench"}><RouterLink to={pdhWbPath} className={classes.keepLink}>{pdhUrl}</RouterLink></Tooltip> :
        <></>;
});

const KeepUrlPath = withStyles(styles)(({auth, res, pdh, classes}: KeepUrlProps & WithStyles<CssRules>) => {
    const keepUrl = getKeepUrl(res, pdh);
    const keepUrlParts = keepUrl ? keepUrl.split('/') : [];
    const keepUrlPath = keepUrlParts.length > 1 ? keepUrlParts.slice(1).join('/') : '';

    const keepUrlPathNav = getKeepNavUrl(auth, res, pdh);
    return keepUrlPathNav ?
        <Tooltip title={"View in keep-web"}><a className={classes.keepLink} href={keepUrlPathNav} target="_blank" rel="noopener noreferrer">{keepUrlPath || '/'}</a></Tooltip> :
        <EmptyValue />;
});

const getKeepNavUrl = (auth: AuthState, file: File | Directory, pdh?: string): string => {
    let keepUrl = getKeepUrl(file, pdh);
    return (getInlineFileUrl(`${auth.config.keepWebServiceUrl}/c=${keepUrl}?api_token=${auth.apiToken}`, auth.config.keepWebServiceUrl, auth.config.keepWebInlineServiceUrl));
};

const getImageUrl = (auth: AuthState, file: File, pdh?: string): string => {
    const keepUrl = getKeepUrl(file, pdh);
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
    if (isExternalValue(directory)) {return {display: <UnsupportedValue />}}

    const normalizedDirectory = normalizeDirectoryLocation(directory);
    return {
        display: <KeepUrlPath auth={auth} res={normalizedDirectory} pdh={pdh}/>,
        collection: <KeepUrlBase auth={auth} res={normalizedDirectory} pdh={pdh}/>,
    };
};

const fileToProcessIOValue = (file: File, secondary: boolean, auth: AuthState, pdh: string | undefined, mainFilePdh: string): ProcessIOValue => {
    if (isExternalValue(file)) {return {display: <UnsupportedValue />}}

    const resourcePdh = getResourcePdhUrl(file, pdh);
    return {
        display: <KeepUrlPath auth={auth} res={file} pdh={pdh}/>,
        secondary,
        imageUrl: isFileImage(file.basename) ? getImageUrl(auth, file, pdh) : undefined,
        collection: (resourcePdh !== mainFilePdh) ? <KeepUrlBase auth={auth} res={file} pdh={pdh}/> : <></>,
    }
};

const isExternalValue = (val: any) =>
    Object.keys(val).includes('$import') ||
    Object.keys(val).includes('$include')

const EmptyValue = withStyles(styles)(
    ({classes}: WithStyles<CssRules>) => <span className={classes.emptyValue}>No value</span>
);

const UnsupportedValue = withStyles(styles)(
    ({classes}: WithStyles<CssRules>) => <span className={classes.emptyValue}>Cannot display value</span>
);

const UnsupportedValueChip = withStyles(styles)(
    ({classes}: WithStyles<CssRules>) => <Chip icon={<InfoIcon />} label={"Cannot display value"} />
);

const ImagePlaceholder = withStyles(styles)(
    ({classes}: WithStyles<CssRules>) => <span className={classes.imagePlaceholder}><ImageIcon /></span>
);
