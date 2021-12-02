// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { connect } from 'react-redux';
import { FixedSizeList } from "react-window";
import AutoSizer from "react-virtualized-auto-sizer";
import servicesProvider from 'common/service-provider';
import { CustomizeTableIcon, DownloadIcon } from 'components/icon/icon';
import { SearchInput } from 'components/search-input/search-input';
import { ListItemIcon, StyleRulesCallback, Theme, WithStyles, withStyles, Tooltip, IconButton, Checkbox, CircularProgress, Button } from '@material-ui/core';
import { FileTreeData } from '../file-tree/file-tree-data';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { RootState } from 'store/store';
import { WebDAV, WebDAVRequestConfig } from 'common/webdav';
import { AuthState } from 'store/auth/auth-reducer';
import { extractFilesData } from 'services/collection-service/collection-service-files-response';
import { DefaultIcon, DirectoryIcon, FileIcon, BackIcon, SidePanelRightArrowIcon } from 'components/icon/icon';
import { setCollectionFiles } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { sortBy } from 'lodash';
import { formatFileSize } from 'common/formatters';
import { getInlineFileUrl, sanitizeToken } from 'views-components/context-menu/actions/helpers';

export interface CollectionPanelFilesProps {
    items: any;
    isWritable: boolean;
    isLoading: boolean;
    tooManyFiles: boolean;
    onUploadDataClick: () => void;
    onSearchChange: (searchValue: string) => void;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>, isWritable: boolean) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, isWritable: boolean) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
    onFileClick: (id: string) => void;
    loadFilesFunc: () => void;
    currentItemUuid: any;
    dispatch: Function;
    collectionPanelFiles: any;
    collectionPanel: any;
}

type CssRules = "backButton" | "backButtonHidden" | "pathPanelPathWrapper" | "uploadButton" | "uploadIcon" | "loader" | "wrapper" | "dataWrapper" | "row" | "rowEmpty" | "leftPanel" | "rightPanel" | "pathPanel" | "pathPanelItem" | "rowName" | "listItemIcon" | "rowActive" | "pathPanelMenu" | "rowSelection" | "leftPanelHidden" | "leftPanelVisible" | "searchWrapper" | "searchWrapperHidden";

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    wrapper: {
        display: 'flex',
        minHeight: '600px',
        marginBottom: '1rem',
        color: 'rgba(0, 0, 0, 0.87)',
        fontSize: '0.875rem',
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontWeight: 400,
        lineHeight: '1.5',
        letterSpacing: '0.01071em'
    },
    backButton: {
        color: '#00bfa5',
        cursor: 'pointer',
        float: 'left',
    },
    backButtonHidden: {
        display: 'none',
    },
    dataWrapper: {
        minHeight: '500px'
    },
    row: {
        display: 'flex',
        marginTop: '0.5rem',
        marginBottom: '0.5rem',
        cursor: 'pointer',
        "&:hover": {
            backgroundColor: 'rgba(0, 0, 0, 0.08)',
        }
    },
    rowEmpty: {
        top: '40%',
        width: '100%',
        textAlign: 'center',
        position: 'absolute'
    },
    loader: {
        top: '50%',
        left: '50%',
        marginTop: '-15px',
        marginLeft: '-15px',
        position: 'absolute'
    },
    rowName: {
        display: 'inline-flex',
        flexDirection: 'column',
        justifyContent: 'center'
    },
    searchWrapper: {
        display: 'inline-block',
        marginBottom: '1rem',
        marginLeft: '1rem',
    },
    searchWrapperHidden: {
        width: '0px'
    },
    rowSelection: {
        padding: '0px',
    },
    rowActive: {
        color: `${theme.palette.primary.main} !important`,
    },
    listItemIcon: {
        display: 'inline-flex',
        flexDirection: 'column',
        justifyContent: 'center'
    },
    pathPanelMenu: {
        float: 'right',
        marginTop: '-15px',
    },
    pathPanel: {
        padding: '1rem',
        marginBottom: '1rem',
        backgroundColor: '#fff',
        boxShadow: '0px 1px 3px 0px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 2px 1px -1px rgb(0 0 0 / 12%)',
    },
    pathPanelPathWrapper: {
        display: 'inline-block',
    },
    leftPanel: {
        flex: 0,
        padding: '1rem',
        marginRight: '1rem',
        whiteSpace: 'nowrap',
        position: 'relative',
        backgroundColor: '#fff',
        boxShadow: '0px 3px 3px 0px rgb(0 0 0 / 20%), 0px 3px 1px 0px rgb(0 0 0 / 14%), 0px 3px 1px -1px rgb(0 0 0 / 12%)',
    },
    leftPanelVisible: {
        opacity: 1,
        flex: '50%',
        animation: `animateVisible 1000ms ${theme.transitions.easing.easeOut}`
    },
    leftPanelHidden: {
        opacity: 0,
        flex: 'initial',
        padding: '0',
        marginRight: '0',
    },
    "@keyframes animateVisible": {
        "0%": {
            opacity: 0,
            flex: 'initial',
        },
        "100%": {
            opacity: 1,
            flex: '50%',
        }
    },
    rightPanel: {
        flex: '50%',
        padding: '1rem',
        paddingTop: '2rem',
        marginTop: '-1rem',
        position: 'relative',
        backgroundColor: '#fff',
        boxShadow: '0px 3px 3px 0px rgb(0 0 0 / 20%), 0px 3px 1px 0px rgb(0 0 0 / 14%), 0px 3px 1px -1px rgb(0 0 0 / 12%)',
    },
    pathPanelItem: {
        cursor: 'pointer',
    },
    uploadIcon: {
        transform: 'rotate(180deg)'
    },
    uploadButton: {
        float: 'right',
    }
});

const pathPromise = {};

export const CollectionPanelFiles = withStyles(styles)(connect((state: RootState) => ({
    auth: state.auth,
    collectionPanel: state.collectionPanel,
    collectionPanelFiles: state.collectionPanelFiles,
}))((props: CollectionPanelFilesProps & WithStyles<CssRules> & { auth: AuthState }) => {
    const { classes, onItemMenuOpen, onUploadDataClick, isWritable, dispatch, collectionPanelFiles, collectionPanel } = props;
    const { apiToken, config } = props.auth;

    const webdavClient = new WebDAV();
    webdavClient.defaults.baseURL = config.keepWebServiceUrl;
    webdavClient.defaults.headers = {
        Authorization: `Bearer ${apiToken}`
    };

    const webDAVRequestConfig: WebDAVRequestConfig = {
        headers: {
            Depth: '1',
        },
    };

    const parentRef = React.useRef(null);
    const [path, setPath]: any = React.useState([]);
    const [pathData, setPathData]: any = React.useState({});
    const [isLoading, setIsLoading] = React.useState(false);
    const [collectionAutofetchEnabled, setCollectionAutofetchEnabled] = React.useState(false);
    const [leftSearch, setLeftSearch] = React.useState('');
    const [rightSearch, setRightSearch] = React.useState('');

    const leftKey = (path.length > 1 ? path.slice(0, path.length - 1) : path).join('/');
    const rightKey = path.join('/');

    const leftData = pathData[leftKey] || [];
    const rightData = pathData[rightKey];

    React.useEffect(() => {
        if (props.currentItemUuid) {
            setPathData({});
            setPath([props.currentItemUuid]);
        }
    }, [props.currentItemUuid]);

    const fetchData = (keys, ignoreCache = false) => {
        const keyArray = Array.isArray(keys) ? keys : [keys];

        Promise.all(keyArray
            .map((key) => {
                const dataExists = !!pathData[key];
                const runningRequest = pathPromise[key];

                if ((!dataExists || ignoreCache) && (!runningRequest || ignoreCache)) {
                    if (!isLoading) {
                        setIsLoading(true);
                    }

                    pathPromise[key] = true;

                    return webdavClient.propfind(`c=${key}`, webDAVRequestConfig);
                }

                return Promise.resolve(null);
            })
            .filter((promise) => !!promise)
        )
            .then((requests) => {
                const newState = requests.map((request, index) => {
                    if (request && request.responseXML != null) {
                        const key = keyArray[index];
                        const result: any = extractFilesData(request.responseXML);
                        const sortedResult = sortBy(result, (n) => n.name).sort((n1, n2) => {
                            if (n1.type === 'directory' && n2.type !== 'directory') {
                                return -1;
                            }
                            if (n1.type !== 'directory' && n2.type === 'directory') {
                                return 1;
                            }
                            return 0;
                        });

                        return { [key]: sortedResult };
                    }
                    return {};
                }).reduce((prev, next) => {
                    return { ...next, ...prev };
                }, {});

                setPathData({ ...pathData, ...newState });
            })
            .finally(() => {
                setIsLoading(false);
                keyArray.forEach(key => delete pathPromise[key]);
            });
    };

    React.useEffect(() => {
        if (rightKey) {
            fetchData(rightKey);
            setLeftSearch('');
            setRightSearch('');
        }
    }, [rightKey]); // eslint-disable-line react-hooks/exhaustive-deps

    React.useEffect(() => {
        const hash = (collectionPanel.item || {}).portableDataHash;

        if (hash && collectionAutofetchEnabled) {
            fetchData([leftKey, rightKey], true);
        }
    }, [(collectionPanel.item || {}).portableDataHash]); // eslint-disable-line react-hooks/exhaustive-deps

    React.useEffect(() => {
        if (rightData) {
            const filtered = rightData.filter(({ name }) => name.indexOf(rightSearch) > -1);
            setCollectionFiles(filtered, false)(dispatch);
        }
    }, [rightData, dispatch, rightSearch]);

    const handleRightClick = React.useCallback(
        (event) => {
            event.preventDefault();
            let elem = event.target;

            while (elem && elem.dataset && !elem.dataset.item) {
                elem = elem.parentNode;
            }

            if (!elem || !elem.dataset) {
                return;
            }

            const { id } = elem.dataset;

            const item: any = {
                id,
                data: rightData.find((elem) => elem.id === id),
            };

            if (id) {
                onItemMenuOpen(event, item, isWritable);

                if (!collectionAutofetchEnabled) {
                    setCollectionAutofetchEnabled(true);
                }
            }
        },
        [onItemMenuOpen, isWritable, rightData] // eslint-disable-line react-hooks/exhaustive-deps
    );

    React.useEffect(() => {
        let node = null;

        if (parentRef && parentRef.current) {
            node = parentRef.current;
            (node as any).addEventListener('contextmenu', handleRightClick);
        }

        return () => {
            if (node) {
                (node as any).removeEventListener('contextmenu', handleRightClick);
            }
        };
    }, [parentRef, handleRightClick]);

    const handleClick = React.useCallback(
        (event: any) => {
            let isCheckbox = false;
            let elem = event.target;

            if (elem.type === 'checkbox') {
                isCheckbox = true;
            }

            while (elem && elem.dataset && !elem.dataset.item) {
                elem = elem.parentNode;
            }

            if (elem && elem.dataset && !isCheckbox) {
                const { parentPath, subfolderPath, breadcrumbPath, type } = elem.dataset;

                if (breadcrumbPath) {
                    const index = path.indexOf(breadcrumbPath);
                    setPath([...path.slice(0, index + 1)]);
                }

                if (parentPath && type === 'directory') {
                    if (path.length > 1) {
                        path.pop()
                    }

                    setPath([...path, parentPath]);
                }

                if (subfolderPath && type === 'directory') {
                    setPath([...path, subfolderPath]);
                }

                if (elem.dataset.id && type === 'file') {
                    const item = rightData.find(({id}) => id === elem.dataset.id) || leftData.find(({ id }) => id === elem.dataset.id);
                    const enhancedItem = servicesProvider.getServices().collectionService.extendFileURL(item);
                    const fileUrl = sanitizeToken(getInlineFileUrl(enhancedItem.url, config.keepWebServiceUrl, config.keepWebInlineServiceUrl), true);
                    window.open(fileUrl, '_blank');
                }
            }

            if (isCheckbox) {
                const { id } = elem.dataset;
                const item = collectionPanelFiles[id];
                props.onSelectionToggle(event, item);
            }
        },
        [path, setPath, collectionPanelFiles] // eslint-disable-line react-hooks/exhaustive-deps
    );

    const getItemIcon = React.useCallback(
        (type: string, activeClass: string | null) => {
            let Icon = DefaultIcon;

            switch (type) {
                case 'directory':
                    Icon = DirectoryIcon;
                    break;
                case 'file':
                    Icon = FileIcon;
                    break;
            }

            return (
                <ListItemIcon className={classNames(classes.listItemIcon, activeClass)}>
                    <Icon />
                </ListItemIcon>
            )
        },
        [classes]
    );

    const getActiveClass = React.useCallback(
        (name) => {
            return path[path.length - 1] === name ? classes.rowActive : null;
        },
        [path, classes]
    );

    const onOptionsMenuOpen = React.useCallback(
        (ev, isWritable) => {
            props.onOptionsMenuOpen(ev, isWritable);
        },
        [props.onOptionsMenuOpen] // eslint-disable-line react-hooks/exhaustive-deps
    );

    return (
        <div data-cy="collection-files-panel" onClick={handleClick} ref={parentRef}>
            <div className={classes.pathPanel}>
                <div className={classes.pathPanelPathWrapper}>
                    {
                        path
                            .map((p: string, index: number) => <span
                                key={`${index}-${p}`}
                                data-item="true"
                                className={classes.pathPanelItem}
                                data-breadcrumb-path={p}
                            >
                                <span className={classes.rowActive}>{index === 0 ? 'Home' : p}</span> <b>/</b>&nbsp;
                            </span>)
                    }
                </div>
                <Tooltip className={classes.pathPanelMenu} title="More options" disableFocusListener>
                    <IconButton
                        data-cy='collection-files-panel-options-btn'
                        onClick={(ev) => {
                            if (!collectionAutofetchEnabled) {
                                setCollectionAutofetchEnabled(true);
                            }
                            onOptionsMenuOpen(ev, isWritable);
                        }}>
                        <CustomizeTableIcon />
                    </IconButton>
                </Tooltip>
            </div>
            <div className={classes.wrapper}>
                <div className={classNames(classes.leftPanel, path.length > 1 ? classes.leftPanelVisible : classes.leftPanelHidden)}>
                    <Tooltip title="Go back" className={path.length > 1 ? classes.backButton : classes.backButtonHidden}>
                        <IconButton onClick={() => setPath([...path.slice(0, path.length -1)])}>
                            <BackIcon />
                        </IconButton>
                    </Tooltip>
                    <div className={path.length > 1 ? classes.searchWrapper : classes.searchWrapperHidden}>
                        <SearchInput selfClearProp={leftKey} label="Search" value={leftSearch} onSearch={setLeftSearch} />
                    </div>
                    <div className={classes.dataWrapper}>
                        {
                            leftData ?
                                <AutoSizer defaultWidth={0}>
                                    {({ height, width }) => {
                                        const filtered = leftData.filter(({ name }) => name.indexOf(leftSearch) > -1);

                                        return !!filtered.length ? <FixedSizeList
                                            height={height}
                                            itemCount={filtered.length}
                                            itemSize={35}
                                            width={width}
                                        >
                                            {
                                                ({ index, style }) => {
                                                    const { id, type, name } = filtered[index];

                                                    return <div
                                                        data-id={id}
                                                        style={style}
                                                        data-item="true"
                                                        data-type={type}
                                                        data-parent-path={name}
                                                        className={classNames(classes.row, getActiveClass(name))}
                                                        key={id}>
                                                            {getItemIcon(type, getActiveClass(name))} 
                                                            <div className={classes.rowName}>
                                                                {name}
                                                            </div>
                                                            {
                                                                getActiveClass(name) ? <SidePanelRightArrowIcon
                                                                    style={{ display: 'inline', marginTop: '5px', marginLeft: '5px' }} /> : null
                                                            }
                                                    </div>;
                                                }
                                            }
                                        </FixedSizeList> : <div className={classes.rowEmpty}>No directories available</div>
                                    }}
                                </AutoSizer> : <div className={classes.row}><CircularProgress className={classes.loader} size={30} /></div>
                        }

                    </div>
                </div>
                <div className={classes.rightPanel}>
                    <div className={classes.searchWrapper}>
                        <SearchInput selfClearProp={rightKey} label="Search" value={rightSearch} onSearch={setRightSearch} />
                    </div>
                    {
                        isWritable &&
                        <Button
                            className={classes.uploadButton}
                            data-cy='upload-button'
                            onClick={() => {
                                if (!collectionAutofetchEnabled) {
                                    setCollectionAutofetchEnabled(true);
                                }
                                onUploadDataClick();
                            }}
                            variant='contained'
                            color='primary'
                            size='small'>
                            <DownloadIcon className={classes.uploadIcon} />
                            Upload data
                        </Button>
                    }
                    <div className={classes.dataWrapper}>
                        {
                            rightData && !isLoading ?
                                <AutoSizer defaultHeight={500}>
                                    {({ height, width }) => {
                                        const filtered = rightData.filter(({ name }) => name.indexOf(rightSearch) > -1);

                                        return !!filtered.length ? <FixedSizeList
                                            height={height}
                                            itemCount={filtered.length}
                                            itemSize={35}
                                            width={width}
                                        >
                                            {
                                                ({ index, style }) => {
                                                    const { id, type, name, size } = filtered[index];

                                                    return <div
                                                        style={style}
                                                        data-id={id}
                                                        data-item="true"
                                                        data-type={type}
                                                        data-subfolder-path={name}
                                                        className={classes.row} key={id}>
                                                        <Checkbox
                                                            color="primary"
                                                            className={classes.rowSelection}
                                                            checked={collectionPanelFiles[id] ? collectionPanelFiles[id].value.selected : false}
                                                        />&nbsp;
                                                    {getItemIcon(type, null)} <div className={classes.rowName}>
                                                            {name}
                                                        </div>
                                                        <span className={classes.rowName} style={{ marginLeft: 'auto', marginRight: '1rem' }}>
                                                            {formatFileSize(size)}
                                                        </span>
                                                    </div>
                                                }
                                            }
                                        </FixedSizeList> : <div className={classes.rowEmpty}>No data available</div>
                                    }}
                                </AutoSizer> : <div className={classes.row}><CircularProgress className={classes.loader} size={30} /></div>
                        }
                    </div>
                </div>
            </div>
        </div>
    );
}));
