// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement, memo } from "react";
import { Dispatch } from "redux";
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
    Table,
    TableHead,
    TableBody,
    TableRow,
    TableCell,
    Paper,
    Grid,
    Chip,
    CircularProgress,
} from "@material-ui/core";
import { ArvadosTheme } from "common/custom-theme";
import { CloseIcon, InputIcon, OutputIcon, MaximizeIcon, UnMaximizeIcon, InfoIcon } from "components/icon/icon";
import { MPVPanelProps } from "components/multi-panel-view/multi-panel-view";
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
    isSecret,
    StringArrayCommandInputParameter,
    StringCommandInputParameter,
    getEnumType,
} from "models/workflow";
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";
import { File } from "models/workflow";
import { getInlineFileUrl } from "views-components/context-menu/actions/helpers";
import { AuthState } from "store/auth/auth-reducer";
import mime from "mime";
import { DefaultView } from "components/default-view/default-view";
import { getNavUrl } from "routes/routes";
import { Link as RouterLink } from "react-router-dom";
import { Link as MuiLink } from "@material-ui/core";
import { InputCollectionMount } from "store/processes/processes-actions";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ProcessOutputCollectionFiles } from "./process-output-collection-files";
import { Process } from "store/processes/process";
import { navigateTo } from "store/navigation/navigation-action";
import classNames from "classnames";
import { DefaultVirtualCodeSnippet } from "components/default-code-snippet/default-virtual-code-snippet";
import { KEEP_URL_REGEX } from "models/resource";
import { FixedSizeList } from 'react-window';
import AutoSizer from "react-virtualized-auto-sizer";
import { LinkProps } from "@material-ui/core/Link";
import { ConditionalTabs } from "components/conditional-tabs/conditional-tabs";

type CssRules =
    | "card"
    | "content"
    | "title"
    | "header"
    | "avatar"
    | "iconHeader"
    | "tableWrapper"
    | "virtualListTableRoot"
    | "parameterTableRoot"
    | "inputMountTableRoot"
    | "virtualListCellText"
    | "jsonWrapper"
    | "keepLink"
    | "collectionLink"
    | "secondaryVal"
    | "emptyValue"
    | "noBorderRow"
    | "symmetricTabs"
    | "wrapTooltip";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: "100%",
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: 0,
    },
    iconHeader: {
        fontSize: "1.875rem",
        color: theme.customs.colors.greyL,
    },
    avatar: {
        alignSelf: "flex-start",
        paddingTop: theme.spacing.unit * 0.5,
    },
    // Card content
    content: {
        height: `calc(100% - ${theme.spacing.unit * 6}px)`,
        padding: theme.spacing.unit * 1.0,
        paddingTop: 0,
        "&:last-child": {
            paddingBottom: theme.spacing.unit * 1,
        },
    },
    // Card title
    title: {
        overflow: "hidden",
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.customs.colors.greyD,
        fontSize: "1.875rem",
    },
    // Applies to parameters / input collection virtual lists and output collection DE
    tableWrapper: {
        height: "auto",
        maxHeight: `calc(100% - ${theme.spacing.unit * 6}px)`,
        overflow: "auto",
        // Use flexbox to keep scrolling at the virtual list level
        display: "flex",
        flexDirection: "column",
        alignItems: "stretch", // Stretches output collection to full width

    },
    // Parameters / input collection virtual list table styles
    virtualListTableRoot: {
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
        // Flex header
        "& thead tr": {
            alignItems: "end",
            "& th": {
                padding: "4px 25px 10px",
            },
        },
        "& tbody": {
            height: "100vh", // Must be constrained by panel maxHeight
        },
        // Flex header/body rows
        "& thead tr, & > tbody tr": {
            display: "flex",
            // Flex header/body cells
            "& th, & td": {
                flexGrow: 1,
                flexShrink: 1,
                flexBasis: 0,
                overflow: "hidden",
            },
        },
        // Flex body rows
        "& tbody tr": {
            height: "40px",
            // Flex body cells
            "& td": {
                padding: "2px 25px 2px",
                overflow: "hidden",
                display: "flex",
                flexDirection: "row",
                alignItems: "center",
                whiteSpace: "nowrap",
            },
        },
    },
    // Parameter tab table column widths
    parameterTableRoot: {
        "& thead tr, & > tbody tr": {
            // ID
            "& th:nth-of-type(1), & td:nth-of-type(1)": {
                flexGrow: 0.7,
            },
            // Collection
            "& th:nth-last-of-type(1), & td:nth-last-of-type(1)": {
                flexGrow: 2,
            },
        },
    },
    // Input collection tab table column widths
    inputMountTableRoot: {
        "& thead tr, & > tbody tr": {
            // Path
            "& th:nth-of-type(1), & td:nth-of-type(1)": {
                flexGrow: 1,
            },
            // PDH
            "& th:nth-last-of-type(1), & td:nth-last-of-type(1)": {
                flexGrow: 0.8,
            },
        },
    },
    // Param value cell typography styles
    virtualListCellText: {
        overflow: "hidden",
        display: "flex",
        // Every cell contents requires a wrapper for the ellipsis
        // since adding ellipses to an anchor element parent results in misaligned tooltip
        "& a, & span": {
            overflow: "hidden",
            textOverflow: "ellipsis",
        },
        '& pre': {
            margin: 0,
            overflow: "hidden",
            textOverflow: "ellipsis",
        },
    },
    // JSON tab wrapper
    jsonWrapper: {
        height: `calc(100% - ${theme.spacing.unit * 6}px)`,
    },
    keepLink: {
        color: theme.palette.primary.main,
        textDecoration: "none",
        // Overflow wrap for mounts table
        overflowWrap: "break-word",
        cursor: "pointer",
    },
    // Output collection tab link
    collectionLink: {
        margin: "10px",
        "& a": {
            color: theme.palette.primary.main,
            textDecoration: "none",
            overflowWrap: "break-word",
            cursor: "pointer",
        },
    },
    secondaryVal: {
        paddingLeft: "20px",
    },
    emptyValue: {
        color: theme.customs.colors.grey700,
    },
    noBorderRow: {
        "& td": {
            borderBottom: "none",
            paddingTop: "2px",
            paddingBottom: "2px",
        },
        height: "24px",
    },
    symmetricTabs: {
        "& button": {
            flexBasis: "0",
        },
    },
    wrapTooltip: {
        maxWidth: "600px",
        wordWrap: "break-word",
    },
});

export enum ProcessIOCardType {
    INPUT = "Input Parameters",
    OUTPUT = "Output Parameters",
}
export interface ProcessIOCardDataProps {
    process?: Process;
    label: ProcessIOCardType;
    params: ProcessIOParameter[] | null;
    raw: any;
    mounts?: InputCollectionMount[];
    outputUuid?: string;
    forceShowParams?: boolean;
}

type ProcessIOCardProps = ProcessIOCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessIOCard = withStyles(styles)(
    ({
        classes,
        label,
        params,
        raw,
        mounts,
        outputUuid,
        doHidePanel,
        doMaximizePanel,
        doUnMaximizePanel,
        panelMaximized,
        panelName,
        process,
        forceShowParams,
    }: ProcessIOCardProps) => {
        const PanelIcon = label === ProcessIOCardType.INPUT ? InputIcon : OutputIcon;
        const mainProcess = !(process && process!.containerRequest.requestingContainerUuid);
        const showParamTable = mainProcess || forceShowParams;

        const loading = raw === null || raw === undefined || params === null;

        const hasRaw = !!(raw && Object.keys(raw).length > 0);
        const hasParams = !!(params && params.length > 0);
        // isRawLoaded allows subprocess panel to display raw even if it's {}
        const isRawLoaded = !!(raw && Object.keys(raw).length >= 0);

        // Subprocess
        const hasInputMounts = !!(label === ProcessIOCardType.INPUT && mounts && mounts.length);
        const hasOutputCollecton = !!(label === ProcessIOCardType.OUTPUT && outputUuid);
        // Subprocess should not show loading if hasOutputCollection or hasInputMounts
        const subProcessLoading = loading && !hasOutputCollecton && !hasInputMounts;

        return (
            <Card
                className={classes.card}
                data-cy="process-io-card"
            >
                <CardHeader
                    className={classes.header}
                    classes={{
                        content: classes.title,
                        avatar: classes.avatar,
                    }}
                    avatar={<PanelIcon className={classes.iconHeader} />}
                    title={
                        <Typography
                            noWrap
                            variant="h6"
                            color="inherit"
                        >
                            {label}
                        </Typography>
                    }
                    action={
                        <div>
                            {doUnMaximizePanel && panelMaximized && (
                                <Tooltip
                                    title={`Unmaximize ${panelName || "panel"}`}
                                    disableFocusListener
                                >
                                    <IconButton onClick={doUnMaximizePanel}>
                                        <UnMaximizeIcon />
                                    </IconButton>
                                </Tooltip>
                            )}
                            {doMaximizePanel && !panelMaximized && (
                                <Tooltip
                                    title={`Maximize ${panelName || "panel"}`}
                                    disableFocusListener
                                >
                                    <IconButton onClick={doMaximizePanel}>
                                        <MaximizeIcon />
                                    </IconButton>
                                </Tooltip>
                            )}
                            {doHidePanel && (
                                <Tooltip
                                    title={`Close ${panelName || "panel"}`}
                                    disableFocusListener
                                >
                                    <IconButton
                                        disabled={panelMaximized}
                                        onClick={doHidePanel}
                                    >
                                        <CloseIcon />
                                    </IconButton>
                                </Tooltip>
                            )}
                        </div>
                    }
                />
                <CardContent className={classes.content}>
                    {showParamTable ? (
                        <>
                            {/* raw is undefined until params are loaded */}
                            {loading && (
                                <Grid
                                    container
                                    item
                                    alignItems="center"
                                    justify="center"
                                >
                                    <CircularProgress />
                                </Grid>
                            )}
                            {/* Once loaded, either raw or params may still be empty
                                *   Raw when all params are empty
                                *   Params when raw is provided by containerRequest properties but workflow mount is absent for preview
                                */}
                            {!loading && (hasRaw || hasParams) && (
                                <ConditionalTabs
                                    variant="fullWidth"
                                    className={classes.symmetricTabs}
                                    tabs={[
                                        {
                                            // params will be empty on processes without workflow definitions in mounts, so we only show raw
                                            show: hasParams,
                                            label: "Parameters",
                                            content: <ProcessIOPreview
                                                    data={params || []}
                                                    valueLabel={forceShowParams ? "Default value" : "Value"}
                                            />,
                                        },
                                        {
                                            show: !forceShowParams,
                                            label: "JSON",
                                            content: <ProcessIORaw data={raw} />,
                                        },
                                        {
                                            show: hasOutputCollecton,
                                            label: "Collection",
                                            content: <ProcessOutputCollection outputUuid={outputUuid} />,
                                        },
                                    ]}
                                />
                            )}
                            {!loading && !hasRaw && !hasParams && (
                                <Grid
                                    container
                                    item
                                    alignItems="center"
                                    justify="center"
                                >
                                    <DefaultView messages={["No parameters found"]} />
                                </Grid>
                            )}
                        </>
                    ) : (
                        // Subprocess
                        <>
                            {subProcessLoading ? (
                                <Grid
                                    container
                                    item
                                    alignItems="center"
                                    justify="center"
                                >
                                    <CircularProgress />
                                </Grid>
                            ) : !subProcessLoading && (hasInputMounts || hasOutputCollecton || isRawLoaded) ? (
                                <ConditionalTabs
                                    variant="fullWidth"
                                    className={classes.symmetricTabs}
                                    tabs={[
                                        {
                                            show: hasInputMounts,
                                            label: "Collections",
                                            content: <ProcessInputMounts mounts={mounts || []} />,
                                        },
                                        {
                                            show: hasOutputCollecton,
                                            label: "Collection",
                                            content: <ProcessOutputCollection outputUuid={outputUuid} />,
                                        },
                                        {
                                            show: isRawLoaded,
                                            label: "JSON",
                                            content: <ProcessIORaw data={raw} />,
                                        },
                                    ]}
                                />
                            ) : (
                                <Grid
                                    container
                                    item
                                    alignItems="center"
                                    justify="center"
                                >
                                    <DefaultView messages={["No data to display"]} />
                                </Grid>
                            )}
                        </>
                    )}
                </CardContent>
            </Card>
        );
    }
);

export type ProcessIOValue = {
    display: ReactElement<any, any>;
    imageUrl?: string;
    collection?: ReactElement<any, any>;
    secondary?: boolean;
};

export type ProcessIOParameter = {
    id: string;
    label: string;
    value: ProcessIOValue;
};

interface ProcessIOPreviewDataProps {
    data: ProcessIOParameter[];
    valueLabel: string;
    hidden?: boolean;
}

type ProcessIOPreviewProps = ProcessIOPreviewDataProps & WithStyles<CssRules>;

const ProcessIOPreview = memo(
    withStyles(styles)(({ data, valueLabel, hidden, classes }: ProcessIOPreviewProps) => {
        const showLabel = data.some((param: ProcessIOParameter) => param.label);

        const hasMoreValues = (index: number) => (
            data[index+1] && !isMainRow(data[index+1])
        );

        const isMainRow = (param: ProcessIOParameter) => (
            param &&
            ((param.id || param.label) &&
            !param.value.secondary)
        );

        const RenderRow = ({index, style}) => {
            const param = data[index];

            const rowClasses = {
                [classes.noBorderRow]: hasMoreValues(index),
            };

            return <TableRow
                style={style}
                className={classNames(rowClasses)}
                data-cy={isMainRow(param) ? "process-io-param" : ""}>
                <TableCell>
                    <Tooltip title={param.id}>
                        <Typography className={classes.virtualListCellText}>
                            <span>
                                {param.id}
                            </span>
                        </Typography>
                    </Tooltip>
                </TableCell>
                {showLabel && <TableCell>
                    <Tooltip title={param.label}>
                        <Typography className={classes.virtualListCellText}>
                            <span>
                                {param.label}
                            </span>
                        </Typography>
                    </Tooltip>
                </TableCell>}
                <TableCell>
                    <ProcessValuePreview
                        value={param.value}
                    />
                </TableCell>
                <TableCell>
                    <Typography className={classes.virtualListCellText}>
                        {/** Collection is an anchor so doesn't require wrapper element */}
                        {param.value.collection}
                    </Typography>
                </TableCell>
            </TableRow>;
        };

        return <div className={classes.tableWrapper} hidden={hidden}>
            <Table
                className={classNames(classes.virtualListTableRoot, classes.parameterTableRoot)}
                aria-label="Process IO Preview"
            >
                <TableHead>
                    <TableRow>
                        <TableCell>Name</TableCell>
                        {showLabel && <TableCell>Label</TableCell>}
                        <TableCell>{valueLabel}</TableCell>
                        <TableCell>Collection</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    <AutoSizer>
                        {({ height, width }) =>
                            <FixedSizeList
                                height={height}
                                itemCount={data.length}
                                itemSize={40}
                                width={width}
                            >
                                {RenderRow}
                            </FixedSizeList>
                        }
                    </AutoSizer>
                </TableBody>
            </Table>
        </div>;
    })
);

interface ProcessValuePreviewProps {
    value: ProcessIOValue;
}

const ProcessValuePreview = withStyles(styles)(({ value, classes }: ProcessValuePreviewProps & WithStyles<CssRules>) => (
    <Typography className={classNames(classes.virtualListCellText, value.secondary && classes.secondaryVal)}>
        {value.display}
    </Typography>
));

interface ProcessIORawDataProps {
    data: ProcessIOParameter[];
    hidden?: boolean;
}

const ProcessIORaw = withStyles(styles)(({ data, hidden, classes }: ProcessIORawDataProps & WithStyles<CssRules>) => (
    <div className={classes.jsonWrapper} hidden={hidden}>
        <Paper elevation={0} style={{minWidth: "100%", height: "100%"}}>
            <DefaultVirtualCodeSnippet
                lines={JSON.stringify(data, null, 2).split('\n')}
                linked
                copyButton
            />
        </Paper>
    </div>
));

interface ProcessInputMountsDataProps {
    mounts: InputCollectionMount[];
    hidden?: boolean;
}

type ProcessInputMountsProps = ProcessInputMountsDataProps & WithStyles<CssRules>;

const ProcessInputMounts = withStyles(styles)(
    connect((state: RootState) => ({
        auth: state.auth,
    }))(({ mounts, hidden, classes, auth }: ProcessInputMountsProps & { auth: AuthState }) => {

        const RenderRow = ({index, style}) => {
            const mount = mounts[index];

            return <TableRow
                key={mount.path}
                style={style}>
                <TableCell>
                    <Tooltip title={mount.path}>
                        <Typography className={classes.virtualListCellText}>
                            <pre>{mount.path}</pre>
                        </Typography>
                    </Tooltip>
                </TableCell>
                <TableCell>
                    <Tooltip title={mount.pdh}>
                        <Typography className={classes.virtualListCellText}>
                            <RouterLink
                                to={getNavUrl(mount.pdh, auth)}
                                className={classes.keepLink}
                            >
                                {mount.pdh}
                            </RouterLink>
                        </Typography>
                    </Tooltip>
                </TableCell>
            </TableRow>;
        };

        return <div className={classes.tableWrapper} hidden={hidden}>
            <Table
                className={classNames(classes.virtualListTableRoot, classes.inputMountTableRoot)}
                aria-label="Process Input Mounts"
                hidden={hidden}
            >
                <TableHead>
                    <TableRow>
                        <TableCell>Path</TableCell>
                        <TableCell>Portable Data Hash</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    <AutoSizer>
                        {({ height, width }) =>
                            <FixedSizeList
                                height={height}
                                itemCount={mounts.length}
                                itemSize={40}
                                width={width}
                            >
                                {RenderRow}
                            </FixedSizeList>
                        }
                    </AutoSizer>
                </TableBody>
            </Table>
        </div>;
    })
);

export interface ProcessOutputCollectionActionProps {
    navigateTo: (uuid: string) => void;
}

const mapNavigateToProps = (dispatch: Dispatch): ProcessOutputCollectionActionProps => ({
    navigateTo: uuid => dispatch<any>(navigateTo(uuid)),
});

type ProcessOutputCollectionProps = {outputUuid: string | undefined, hidden?: boolean} & ProcessOutputCollectionActionProps &  WithStyles<CssRules>;

const ProcessOutputCollection = withStyles(styles)(connect(null, mapNavigateToProps)(({ outputUuid, hidden, navigateTo, classes }: ProcessOutputCollectionProps) => (
    <div className={classes.tableWrapper} hidden={hidden}>
        <>
            {outputUuid && (
                <Typography className={classes.collectionLink}>
                    Output Collection:{" "}
                    <MuiLink
                        className={classes.keepLink}
                        onClick={() => {
                            navigateTo(outputUuid || "");
                        }}
                    >
                        {outputUuid}
                    </MuiLink>
                </Typography>
            )}
            <ProcessOutputCollectionFiles
                isWritable={false}
                currentItemUuid={outputUuid}
            />
        </>
    </div>
)));

type FileWithSecondaryFiles = {
    secondaryFiles: File[];
};

export const getIOParamDisplayValue = (auth: AuthState, input: CommandInputParameter | CommandOutputParameter, pdh?: string): ProcessIOValue[] => {
    switch (true) {
        case isSecret(input):
            return [{ display: <SecretValue /> }];

        case isPrimitiveOfType(input, CWLType.BOOLEAN):
            const boolValue = (input as BooleanCommandInputParameter).value;
            return boolValue !== undefined && !(Array.isArray(boolValue) && boolValue.length === 0)
                ? [{ display: <PrimitiveTooltip data={boolValue}>{renderPrimitiveValue(boolValue, false)}</PrimitiveTooltip> }]
                : [{ display: <EmptyValue /> }];

        case isPrimitiveOfType(input, CWLType.INT):
        case isPrimitiveOfType(input, CWLType.LONG):
            const intValue = (input as IntCommandInputParameter).value;
            return intValue !== undefined &&
                // Missing values are empty array
                !(Array.isArray(intValue) && intValue.length === 0)
                ? [{ display: <PrimitiveTooltip data={intValue}>{renderPrimitiveValue(intValue, false)}</PrimitiveTooltip> }]
                : [{ display: <EmptyValue /> }];

        case isPrimitiveOfType(input, CWLType.FLOAT):
        case isPrimitiveOfType(input, CWLType.DOUBLE):
            const floatValue = (input as FloatCommandInputParameter).value;
            return floatValue !== undefined && !(Array.isArray(floatValue) && floatValue.length === 0)
                ? [{ display: <PrimitiveTooltip data={floatValue}>{renderPrimitiveValue(floatValue, false)}</PrimitiveTooltip> }]
                : [{ display: <EmptyValue /> }];

        case isPrimitiveOfType(input, CWLType.STRING):
            const stringValue = (input as StringCommandInputParameter).value || undefined;
            return stringValue !== undefined && !(Array.isArray(stringValue) && stringValue.length === 0)
                ? [{ display: <PrimitiveTooltip data={stringValue}>{renderPrimitiveValue(stringValue, false)}</PrimitiveTooltip> }]
                : [{ display: <EmptyValue /> }];

        case isPrimitiveOfType(input, CWLType.FILE):
            const mainFile = (input as FileCommandInputParameter).value;
            // secondaryFiles: File[] is not part of CommandOutputParameter so we cast to access secondaryFiles
            const secondaryFiles = (mainFile as unknown as FileWithSecondaryFiles)?.secondaryFiles || [];
            const files = [...(mainFile && !(Array.isArray(mainFile) && mainFile.length === 0) ? [mainFile] : []), ...secondaryFiles];
            const mainFilePdhUrl = mainFile ? getResourcePdhUrl(mainFile, pdh) : "";
            return files.length
                ? files.map((file, i) => fileToProcessIOValue(file, i > 0, auth, pdh, i > 0 ? mainFilePdhUrl : ""))
                : [{ display: <EmptyValue /> }];

        case isPrimitiveOfType(input, CWLType.DIRECTORY):
            const directory = (input as DirectoryCommandInputParameter).value;
            return directory !== undefined && !(Array.isArray(directory) && directory.length === 0)
                ? [directoryToProcessIOValue(directory, auth, pdh)]
                : [{ display: <EmptyValue /> }];

        case getEnumType(input) !== null:
            const enumValue = (input as EnumCommandInputParameter).value;
            return enumValue !== undefined && enumValue ? [{ display: <PrimitiveTooltip data={enumValue}>{enumValue}</PrimitiveTooltip> }] : [{ display: <EmptyValue /> }];

        case isArrayOfType(input, CWLType.STRING):
            const strArray = (input as StringArrayCommandInputParameter).value || [];
            return strArray.length ? [{ display: <PrimitiveArrayTooltip data={strArray}>{strArray.map(val => renderPrimitiveValue(val, true))}</PrimitiveArrayTooltip> }] : [{ display: <EmptyValue /> }];

        case isArrayOfType(input, CWLType.INT):
        case isArrayOfType(input, CWLType.LONG):
            const intArray = (input as IntArrayCommandInputParameter).value || [];
            return intArray.length ? [{ display: <PrimitiveArrayTooltip data={intArray}>{intArray.map(val => renderPrimitiveValue(val, true))}</PrimitiveArrayTooltip> }] : [{ display: <EmptyValue /> }];

        case isArrayOfType(input, CWLType.FLOAT):
        case isArrayOfType(input, CWLType.DOUBLE):
            const floatArray = (input as FloatArrayCommandInputParameter).value || [];
            return floatArray.length ? [{ display: <PrimitiveArrayTooltip data={floatArray}>{floatArray.map(val => renderPrimitiveValue(val, true))}</PrimitiveArrayTooltip> }] : [{ display: <EmptyValue /> }];

        case isArrayOfType(input, CWLType.FILE):
            const fileArrayMainFiles = (input as FileArrayCommandInputParameter).value || [];
            const firstMainFilePdh = fileArrayMainFiles.length > 0 && fileArrayMainFiles[0] ? getResourcePdhUrl(fileArrayMainFiles[0], pdh) : "";

            // Convert each main and secondaryFiles into array of ProcessIOValue preserving ordering
            let fileArrayValues: ProcessIOValue[] = [];
            for (let i = 0; i < fileArrayMainFiles.length; i++) {
                const secondaryFiles = (fileArrayMainFiles[i] as unknown as FileWithSecondaryFiles)?.secondaryFiles || [];
                fileArrayValues.push(
                    // Pass firstMainFilePdh to secondary files and every main file besides the first to hide pdh if equal
                    ...(fileArrayMainFiles[i] ? [fileToProcessIOValue(fileArrayMainFiles[i], false, auth, pdh, i > 0 ? firstMainFilePdh : "")] : []),
                    ...secondaryFiles.map(file => fileToProcessIOValue(file, true, auth, pdh, firstMainFilePdh))
                );
            }

            return fileArrayValues.length ? fileArrayValues : [{ display: <EmptyValue /> }];

        case isArrayOfType(input, CWLType.DIRECTORY):
            const directories = (input as DirectoryArrayCommandInputParameter).value || [];
            return directories.length ? directories.map(directory => directoryToProcessIOValue(directory, auth, pdh)) : [{ display: <EmptyValue /> }];

        default:
            return [{ display: <UnsupportedValue /> }];
    }
};

interface PrimitiveTooltipProps {
    data: boolean | number | string;
}

const PrimitiveTooltip = (props: React.PropsWithChildren<PrimitiveTooltipProps>) => (
    <Tooltip title={typeof props.data !== 'object' ? String(props.data) : ""}>
        <pre>{props.children}</pre>
    </Tooltip>
);

interface PrimitiveArrayTooltipProps {
    data: string[];
}

const PrimitiveArrayTooltip = (props: React.PropsWithChildren<PrimitiveArrayTooltipProps>) => (
    <Tooltip title={props.data.join(', ')}>
        <span>{props.children}</span>
    </Tooltip>
);


const renderPrimitiveValue = (value: any, asChip: boolean) => {
    const isObject = typeof value === "object";
    if (!isObject) {
        return asChip ? (
            <Chip
                key={value}
                label={String(value)}
                style={{marginRight: "10px"}}
            />
        ) : (
            <>{String(value)}</>
        );
    } else {
        return asChip ? <UnsupportedValueChip /> : <UnsupportedValue />;
    }
};

/*
 * @returns keep url without keep: prefix
 */
const getKeepUrl = (file: File | Directory, pdh?: string): string => {
    const isKeepUrl = file.location?.startsWith("keep:") || false;
    const keepUrl = isKeepUrl ? file.location?.replace("keep:", "") : pdh ? `${pdh}/${file.location}` : file.location;
    return keepUrl || "";
};

interface KeepUrlProps {
    auth: AuthState;
    res: File | Directory;
    pdh?: string;
}

const getResourcePdhUrl = (res: File | Directory, pdh?: string): string => {
    const keepUrl = getKeepUrl(res, pdh);
    return keepUrl ? keepUrl.split("/").slice(0, 1)[0] : "";
};

const KeepUrlBase = withStyles(styles)(({ auth, res, pdh, classes }: KeepUrlProps & WithStyles<CssRules>) => {
    const pdhUrl = getResourcePdhUrl(res, pdh);
    // Passing a pdh always returns a relative wb2 collection url
    const pdhWbPath = getNavUrl(pdhUrl, auth);
    return pdhUrl && pdhWbPath ? (
        <Tooltip title={<>View collection in Workbench<br />{pdhUrl}</>}>
            <RouterLink
                to={pdhWbPath}
                className={classes.keepLink}
            >
                {pdhUrl}
            </RouterLink>
        </Tooltip>
    ) : (
        <></>
    );
});

const KeepUrlPath = withStyles(styles)(({ auth, res, pdh, classes }: KeepUrlProps & WithStyles<CssRules>) => {
    const keepUrl = getKeepUrl(res, pdh);
    const keepUrlParts = keepUrl ? keepUrl.split("/") : [];
    const keepUrlPath = keepUrlParts.length > 1 ? keepUrlParts.slice(1).join("/") : "";

    const keepUrlPathNav = getKeepNavUrl(auth, res, pdh);
    return keepUrlPathNav ? (
        <Tooltip classes={{tooltip: classes.wrapTooltip}} title={<>View in keep-web<br />{keepUrlPath || "/"}</>}>
            <a
                className={classes.keepLink}
                href={keepUrlPathNav}
                target="_blank"
                rel="noopener noreferrer"
            >
                {keepUrlPath || "/"}
            </a>
        </Tooltip>
    ) : (
        <EmptyValue />
    );
});

const getKeepNavUrl = (auth: AuthState, file: File | Directory, pdh?: string): string => {
    let keepUrl = getKeepUrl(file, pdh);
    return getInlineFileUrl(
        `${auth.config.keepWebServiceUrl}/c=${keepUrl}?api_token=${auth.apiToken}`,
        auth.config.keepWebServiceUrl,
        auth.config.keepWebInlineServiceUrl
    );
};

const getImageUrl = (auth: AuthState, file: File, pdh?: string): string => {
    const keepUrl = getKeepUrl(file, pdh);
    return getInlineFileUrl(
        `${auth.config.keepWebServiceUrl}/c=${keepUrl}?api_token=${auth.apiToken}`,
        auth.config.keepWebServiceUrl,
        auth.config.keepWebInlineServiceUrl
    );
};

const isFileImage = (basename?: string): boolean => {
    return basename ? (mime.getType(basename) || "").startsWith("image/") : false;
};

const isFileUrl = (location?: string): boolean =>
    !!location && !KEEP_URL_REGEX.exec(location) && (location.startsWith("http://") || location.startsWith("https://"));

const normalizeDirectoryLocation = (directory: Directory): Directory => {
    if (!directory.location) {
        return directory;
    }
    return {
        ...directory,
        location: (directory.location || "").endsWith("/") ? directory.location : directory.location + "/",
    };
};

const directoryToProcessIOValue = (directory: Directory, auth: AuthState, pdh?: string): ProcessIOValue => {
    if (isExternalValue(directory)) {
        return { display: <UnsupportedValue /> };
    }

    const normalizedDirectory = normalizeDirectoryLocation(directory);
    return {
        display: (
            <KeepUrlPath
                auth={auth}
                res={normalizedDirectory}
                pdh={pdh}
            />
        ),
        collection: (
            <KeepUrlBase
                auth={auth}
                res={normalizedDirectory}
                pdh={pdh}
            />
        ),
    };
};

type MuiLinkWithTooltipProps = WithStyles<CssRules> & React.PropsWithChildren<LinkProps>;

const MuiLinkWithTooltip = withStyles(styles)((props: MuiLinkWithTooltipProps) => (
    <Tooltip title={props.title} classes={{tooltip: props.classes.wrapTooltip}}>
        <MuiLink {...props}>
            {props.children}
        </MuiLink>
    </Tooltip>
));

const fileToProcessIOValue = (file: File, secondary: boolean, auth: AuthState, pdh: string | undefined, mainFilePdh: string): ProcessIOValue => {
    if (isExternalValue(file)) {
        return { display: <UnsupportedValue /> };
    }

    if (isFileUrl(file.location)) {
        return {
            display: (
                <MuiLinkWithTooltip
                    href={file.location}
                    target="_blank"
                    rel="noopener"
                    title={file.location}
                >
                    {file.location}
                </MuiLinkWithTooltip>
            ),
            secondary,
        };
    }

    const resourcePdh = getResourcePdhUrl(file, pdh);
    return {
        display: (
            <KeepUrlPath
                auth={auth}
                res={file}
                pdh={pdh}
            />
        ),
        secondary,
        imageUrl: isFileImage(file.basename) ? getImageUrl(auth, file, pdh) : undefined,
        collection:
            resourcePdh !== mainFilePdh ? (
                <KeepUrlBase
                    auth={auth}
                    res={file}
                    pdh={pdh}
                />
            ) : (
                <></>
            ),
    };
};

const isExternalValue = (val: any) => Object.keys(val).includes("$import") || Object.keys(val).includes("$include");

export const EmptyValue = withStyles(styles)(({ classes }: WithStyles<CssRules>) => <span className={classes.emptyValue}>No value</span>);

const UnsupportedValue = withStyles(styles)(({ classes }: WithStyles<CssRules>) => <span className={classes.emptyValue}>Cannot display value</span>);

const SecretValue = withStyles(styles)(({ classes }: WithStyles<CssRules>) => <span className={classes.emptyValue}>Cannot display secret</span>);

const UnsupportedValueChip = withStyles(styles)(({ classes }: WithStyles<CssRules>) => (
    <Chip
        icon={<InfoIcon />}
        label={"Cannot display value"}
    />
));
