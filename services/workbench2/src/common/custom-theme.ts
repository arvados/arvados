// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTheme } from '@mui/material/styles';
import { StyleRulesCallback } from '@mui/styles';
import { DeprecatedThemeOptions, Theme } from '@mui/material/styles';
import { blue, grey, green, yellow, red } from '@mui/material/colors';

export interface ArvadosThemeOptions extends DeprecatedThemeOptions {
    customs: any;
    components?: any;
}

export interface ArvadosTheme extends Theme {
    customs: {
        colors: Colors
    };
}

export type CustomStyleRulesCallback<ClassKey extends string = string> =
    StyleRulesCallback<Theme, {}, ClassKey>

interface Colors {
    green700: string;
    green800: string;
    yellow100: string;
    yellow700: string;
    yellow900: string;
    red100: string;
    red900: string;
    blue500: string;
    blue700: string;
    grey500: string;
    grey600: string;
    grey700: string;
    grey900: string;
    purple: string;
    orange: string;
    darkOrange: string;
    greyL: string;
    greyD: string;
    darkblue: string;
}

/**
* arvadosGreyLight is the hex equivalent of rgba(0,0,0,0.87) on #fafafa background and arvadosGreyDark is the hex equivalent of rgab(0,0,0,0.54) on #fafafa background
*/

const arvadosDarkBlue = '#052a3c';
const arvadosGreyLight = '#737373';
const arvadosGreyVeryLight = '#fafafa';
const arvadosGreyDark = '#212121';
const grey500 = grey["500"];
const grey600 = grey["600"];
const grey700 = grey["700"];
const grey800 = grey["800"];
const grey900 = grey["900"];

export const themeOptions: ArvadosThemeOptions = {
    customs: {
        colors: {
            green700: green["700"],
            green800: green["800"],
            yellow100: yellow["100"],
            yellow700: yellow["700"],
            yellow900: yellow["900"],
            red100: red["100"],
            red900: red['900'],
            blue500: blue['500'],
            blue700: blue['700'],
            grey500: grey500,
            grey600: grey600,
            grey700: grey700,
            grey800: grey800,
            grey900: grey900,
            darkblue: arvadosDarkBlue,
            orange: '#f0ad4e',
            darkOrange: '#9A6E31',
            greyL: arvadosGreyLight,
            greyD: arvadosGreyDark,
        }
    },
    components: {
        MuiTableCell: {
            styleOverrides: {
                root: { paddingTop: '12px', paddingBottom: '12px' }
            },
        },
        MuiTypography: {
            styleOverrides: {
                body1: { fontSize: '0.875rem' }
            },
        },
        MuiAppBar: {
            styleOverrides: {
                colorPrimary: { backgroundColor: arvadosDarkBlue }
            },
        },
        MuiTabs: {
            styleOverrides: {
                root: {
                    color: grey600
                },
                indicator: {
                    backgroundColor: arvadosDarkBlue
                },
            },
        },
        MuiTab: {
            styleOverrides: {
                root: {
                    '&$selected': {
                        fontWeight: 700,
                    },
                },
            },
        },
        MuiList: {
            styleOverrides: {
                root: {
                    color: grey900
                },
            },
        },
        MuiListItem: {
            styleOverrides: {
                root: {
                    color: grey900
                },
            }
        },
        MuiListItemText: {
            styleOverrides: {
                root: {
                    padding: 0,
                    paddingBottom: '2px',
                },
            },
        },
        MuiListItemIcon: {
            styleOverrides: {
                root: {
                    fontSize: '1.25rem',
                    minWidth: 0,
                    marginRight: '16px'
                },
            },
        },
        MuiCardHeader: {
            styleOverrides: {
                avatar: {
                    display: 'flex',
                    alignItems: 'center'
                },
                title: {
                    color: arvadosGreyDark,
                    fontSize: '1.25rem'
                },
            },
        },
        MuiAccordion: {
            styleOverrides: {
                root: {
                    backgroundColor: arvadosGreyVeryLight,
                },
            },
        },
        MuiAccordionDetails: {
            styleOverrides: {
                root: {
                    marginBottom: 0,
                    paddingBottom: '4px',
                },
            },
        },
        MuiAccordionSummary: {
            styleOverrides: {
                content: {
                    '&$expanded': {
                        margin: 0,
                    },
                    color: grey700,
                    fontSize: '1.25rem',
                    margin: 0,
                },
            },
        },
        MuiMenuItem: {
            styleOverrides: {
                root: {
                    padding: '8px 16px'
                },
            },
        },
        MuiInput: {
            styleOverrides: {
                root: {
                    fontSize: '0.875rem'
                },
                underline: {
                    '&:after': {
                        borderBottomColor: arvadosDarkBlue
                    },
                    '&:hover:not($disabled):not($focused):not($error):before': {
                        borderBottom: '1px solid inherit'
                    },
                },
            },
        },
        MuiFormLabel: {
            styleOverrides: {
                root: {
                    fontSize: '0.875rem',
                    "&$focused": {
                        "&$focused:not($error)": {
                            color: arvadosDarkBlue
                        },
                    },
                },
            },
        },
        MuiStepIcon: {
            styleOverrides: {
                root: {
                    '&$active': {
                        color: arvadosDarkBlue
                    },
                    '&$completed': {
                        color: 'inherited'
                    },
                },
            },
        },
        MuiStepConnector: {
            styleOverrides: {
                vertical: {
                    flex: "unset",
                },
            },
        },
        MuiLinearProgress: {
            styleOverrides: {
                barColorSecondary: {
                    backgroundColor: red['700']
                },
            },
        },
    },
    mixins: {
        toolbar: {
            minHeight: '48px'
        }
    },
    palette: {
        primary: {
            main: '#017ead',
            dark: '#015272',
            light: '#82cffd',
            contrastText: '#fff',
        },
        background: {
            default: '#fafafa',
        },
    },
};

export const CustomTheme = createTheme(themeOptions);
