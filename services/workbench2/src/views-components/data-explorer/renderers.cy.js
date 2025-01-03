// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ProcessStatus, renderFileSize, renderMembersCount } from './renderers';
import { Provider } from 'react-redux';
import configureMockStore from 'redux-mock-store'
import { ResourceKind } from '../../models/resource';
import { ContainerRequestState as CR } from '../../models/container-request';
import { ContainerState as C } from '../../models/container';
import { ProcessStatus as PS } from '../../store/processes/process';
import { ThemeProvider, StyledEngineProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';

const middlewares = [];
const mockStore = configureMockStore(middlewares);

describe('renderers', () => {
    let props = null;

    describe('ProcessStatus', () => {
        props = {
            uuid: 'zzzzz-xvhdp-zzzzzzzzzzzzzzz',
            theme: {
                customs: {
                    colors: {
                        // Color values are arbitrary, but they should be
                        // representative of the colors used in the UI.
                        green800: 'rgb(0, 255, 0)',
                        red900: 'rgb(255, 0, 0)',
                        orange: 'rgb(240, 173, 78)',
                        grey600: 'rgb(128, 128, 128)',
                    }
                },
                spacing: (value) => value * 8,
                palette: {
                    common: {
                        white: 'rgb(255, 255, 255)',
                    },
                },
            },
        };

        [
            // CR Status ; Priority ; C Status ; Exit Code ; C RuntimeStatus ; Expected label ; Expected bg color ; Expected fg color
            [CR.COMMITTED, 1, C.RUNNING, null, {}, PS.RUNNING, props.theme.palette.common.white, props.theme.customs.colors.green800],
            [CR.COMMITTED, 1, C.RUNNING, null, { error: 'whoops' }, PS.FAILING, props.theme.palette.common.white, props.theme.customs.colors.red900],
            [CR.COMMITTED, 1, C.RUNNING, null, { warning: 'watch out!' }, PS.WARNING, props.theme.palette.common.white, props.theme.customs.colors.green800],
            [CR.FINAL, 1, C.CANCELLED, null, {}, PS.CANCELLED, props.theme.customs.colors.red900, props.theme.palette.common.white],
            [CR.FINAL, 1, C.COMPLETE, 137, {}, PS.FAILED, props.theme.customs.colors.red900, props.theme.palette.common.white],
            [CR.FINAL, 1, C.COMPLETE, 0, {}, PS.COMPLETED, props.theme.customs.colors.green800, props.theme.palette.common.white],
            [CR.COMMITTED, 0, C.LOCKED, null, {}, PS.ONHOLD, props.theme.customs.colors.grey600, props.theme.palette.common.white],
            [CR.COMMITTED, 0, C.QUEUED, null, {}, PS.ONHOLD, props.theme.customs.colors.grey600, props.theme.palette.common.white],
            [CR.COMMITTED, 1, C.LOCKED, null, {}, PS.QUEUED, props.theme.palette.common.white, props.theme.customs.colors.grey600],
            [CR.COMMITTED, 1, C.QUEUED, null, {}, PS.QUEUED, props.theme.palette.common.white, props.theme.customs.colors.grey600],
        ].forEach(([crState, crPrio, cState, exitCode, rs, eLabel, eColor, tColor]) => {
            it(`should render the state label '${eLabel}' and color '${eColor}' for CR state=${crState}, priority=${crPrio}, C state=${cState}, exitCode=${exitCode} and RuntimeStatus=${JSON.stringify(rs)}`, () => {
                const containerUuid = 'zzzzz-dz642-zzzzzzzzzzzzzzz';
                const store = mockStore({
                    resources: {
                        [props.uuid]: {
                            kind: ResourceKind.CONTAINER_REQUEST,
                            state: crState,
                            containerUuid: containerUuid,
                            priority: crPrio,
                        },
                        [containerUuid]: {
                            kind: ResourceKind.CONTAINER,
                            state: cState,
                            runtimeStatus: rs,
                            exitCode: exitCode,
                        },
                    }
                });

                cy.mount(
                    <Provider store={store}>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessStatus {...props} />
                        </ThemeProvider>
                    </Provider>);

                cy.get('span').should('have.text', eLabel);
                cy.get('span').should('have.css', 'color', tColor);
                cy.get('[data-cy=process-status-chip]').should('have.css', 'background-color', eColor);
            });
        })
    });

    describe('ResourceFileSize', () => {
        it('should render collection fileSizeTotal', () => {
            const resource_1 = {
                uuid: 'UUID_1',
                fileSizeTotal: 100,
            };

            // when
            cy.mount(<p data-cy="file-size">{renderFileSize(resource_1)}</p>);

            // then
            cy.get('[data-cy=file-size]').should('have.text', '100 B');
        });

        it('should render 0 B as file size', () => {
            // given
            const resource_2 = {
                uuid: 'UUID_2',
                fileSizeTotal: 0,
            };

            // when
            cy.mount(<p data-cy="file-size">{renderFileSize(resource_2)}</p>);

            // then
            cy.get('[data-cy=file-size]').should('have.text', '0 B');
        });

        it('should render empty string when fileSizeTotal is undefined', () => {
            const resource_3 = {
                uuid: 'UUID_3',
            };
            
            // when
            cy.mount(<p data-cy="file-size">{renderFileSize(resource_3)}</p>);

            // then
            cy.get('[data-cy=file-size]').should('have.text', '-');
        });
    });

    describe('GroupMembersCount', () => {
        let fakeGroup;
        beforeEach(() => {
            fakeGroup = {
                "canManage": true,
                "canWrite": true,
                "createdAt": "2020-09-24T22:52:57.546521000Z",
                "deleteAt": null,
                "description": "Test Group",
                "etag": "0000000000000000000000000",
                "frozenByUuid": null,
                "groupClass": "role",
                "isTrashed": false,
                "kind": ResourceKind.GROUP,
                "modifiedAt": "2020-09-24T22:52:57.545669000Z",
                "modifiedByUserUuid": "zzzzz-tpzed-000000000000000",
                "name": "System group",
                "ownerUuid": "zzzzz-tpzed-000000000000000",
                "properties": {},
                "trashAt": null,
                "uuid": 'zzzzz-j7d0g-000000000000000',
                "writableBy": [
                    "zzzzz-tpzed-000000000000000",
                ]
            };
        });

        it('shows loading group count when no memberCount', () => {
            cy.mount(
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <p>{renderMembersCount(fakeGroup)}</p>
                    </ThemeProvider>
                </StyledEngineProvider>
            );

            cy.get('[data-testid=three-dots-svg]').should('exist');
        });

        it('shows group count when memberCount present', () => {
            // Given
            fakeGroup = {
                    ...fakeGroup,
                    "memberCount": 765,
                }

            cy.mount(
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <p data-cy="members-count">{renderMembersCount(fakeGroup)}</p>
                    </ThemeProvider>
                </StyledEngineProvider>
            );

            cy.get('[data-cy=members-count]').should('have.text', '765');
        });

        it('shows group count error icon when memberCount is null', () => {
            // Given
            fakeGroup = {
                    ...fakeGroup,
                    "memberCount": null,
                }

            cy.mount(
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <p>{renderMembersCount(fakeGroup)}</p>
                    </ThemeProvider>
                </StyledEngineProvider>
            );

            cy.get('[data-testid=ErrorRoundedIcon]').should('exist');
        });

    });

});
