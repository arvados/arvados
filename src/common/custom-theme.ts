// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createMuiTheme } from '@material-ui/core/styles';
import { ThemeOptions, Theme } from '@material-ui/core/styles/createMuiTheme';
import blue from '@material-ui/core/colors/blue';
import grey from '@material-ui/core/colors/grey';
import green from '@material-ui/core/colors/green';
import yellow from '@material-ui/core/colors/yellow';
import red from '@material-ui/core/colors/red';

export interface ArvadosThemeOptions extends ThemeOptions {
    customs: any;
}

export interface ArvadosTheme extends Theme {
    customs: {
        colors: Colors
    };
}

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
    greyL: string;
    greyD: string;
    darkblue: string;
}

/**
* arvadosGreyLight is the hex equivalent of rgba(0,0,0,0.87) on #fafafa background and arvadosGreyDark is the hex equivalent of rgab(0,0,0,0.54) on #fafafa background  
*/

const arvadosDarkBlue = '#052a3c';
const arvadosGreyLight = '#737373'; 
const arvadosGreyDark = '#212121'; 
const grey500 = grey["500"];
const grey600 = grey["600"];
const grey700 = grey["700"];
const grey800 = grey["800"];
const grey900 = grey["900"];

export const themeOptions: ArvadosThemeOptions = {
    typography: {
        useNextVariants: true,
    },
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
            greyL: arvadosGreyLight,
            greyD: arvadosGreyDark,
        }
    },
    overrides: {
        MuiTypography: {
            body1: {
                fontSize: '0.8125rem'
            }
        },
        MuiAppBar: {
            colorPrimary: {
                backgroundColor: arvadosDarkBlue
            }
        },
        MuiTabs: {
            root: {
                color: grey600
            },
            indicator: {
                backgroundColor: arvadosDarkBlue
            }
        },
        MuiTab: {
            root: {
                '&$selected': {
                    fontWeight: 700,
                }
            }
        },
        MuiList: {
            root: {
                color: grey900
            }
        },
        MuiListItemText: {
            root: {
                padding: 0
            }
        },
        MuiListItemIcon: {
            root: {
                fontSize: '1.25rem',
            }
        },
        MuiCardHeader: {
            avatar: {
                display: 'flex',
                alignItems: 'center'
            },
            title: {
                color: arvadosGreyDark, 
                fontSize: '1.25rem'
            }
        },
        MuiExpansionPanel: {
            expanded: {
                marginTop: '8px',
            }
        },
        MuiExpansionPanelDetails: {
            root: {
                marginBottom: 0,
                paddingBottom: '4px',
            }
        },
        MuiExpansionPanelSummary: {
            content: {
                '&$expanded': {
                    margin: 0,
                },
                color: grey700,
                fontSize: '1.25rem',
                margin: 0,
            },
            expanded: {},
        },
        MuiMenuItem: {
            root: {
                padding: '8px 16px'
            }
        },
        MuiInput: {
            root: {
                fontSize: '0.875rem'
            },
            underline: {
                '&:after': {
                    borderBottomColor: arvadosDarkBlue
                },
                '&:hover:not($disabled):not($focused):not($error):before': {
                    borderBottom: '1px solid inherit'
                }
            }
        },
        MuiFormLabel: {
            root: {
                fontSize: '0.875rem',
                "&$focused": {
                    "&$focused:not($error)": {
                        color: arvadosDarkBlue
                    }
                }
            }
        },
        MuiStepIcon: {
            root: {
                '&$active': {
                    color: arvadosDarkBlue
                },
                '&$completed': {
                    color: 'inherited'
                },
            }
        }
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
            contrastText: '#fff'
        }
    },
};

export const CustomTheme = createMuiTheme(themeOptions);
