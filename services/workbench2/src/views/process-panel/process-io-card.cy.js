// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { combineReducers, createStore } from "redux";
import {
    ThemeProvider,
    StyledEngineProvider,
} from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { Provider } from 'react-redux';
import { ProcessIOCard, ProcessIOCardType } from './process-io-card';
import { MemoryRouter } from 'react-router-dom';

describe('renderers', () => {
    let store;

    describe('ProcessStatus', () => {

        beforeEach(() => {
            store = createStore(combineReducers({
                auth: (state = {}, action) => { return {...state, config: {} } },
                collectionPanel: (state = {}, action) => state,
                collectionPanelFiles: (state = {}, action) => { return {...state, item: { portableDataHash: '12345'} } },
            }));

            // response can be anything not 404
            cy.intercept('*', { foo: 'bar' });

        });

        it('shows main process input loading when raw or params null', () => {
            try {
            // when
                cy.mount(
                    <Provider store={store}>
                        <StyledEngineProvider injectFirst>
                            <ThemeProvider theme={CustomTheme}>
                                <ProcessIOCard
                                    label={ProcessIOCardType.INPUT}
                                    process={false} // Treat as a main process, no requestingContainerUuid
                                    params={null}
                                    raw={{}}
                                />
                            </ThemeProvider>
                        </StyledEngineProvider>
                    </Provider>
                );
                // then
                cy.get('[data-cy=process-io-card]').within(() => {
                    cy.get('[data-cy=conditional-tab]').should('not.exist');
                    cy.get('[data-cy=process-io-circular-progress]').should('exist');
                });
            } catch (error) {
                console.error(error)
            }


            try {
                // when
                cy.mount(
                    <Provider store={store}>
                        <StyledEngineProvider injectFirst>
                            <ThemeProvider theme={CustomTheme}>
                                <ProcessIOCard
                                    label={ProcessIOCardType.INPUT}
                                    process={false} // Treat as a main process, no requestingContainerUuid
                                    params={[]}
                                    raw={null}
                                />
                            </ThemeProvider>
                        </StyledEngineProvider>
                    </Provider>
                );
            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=conditional-tab]').should('not.exist');
                cy.get('[data-cy=process-io-circular-progress]').should('exist');
            });    
            } catch (error) {
                console.error(error)
            }
        });

        it('shows main process empty params and raw', () => {
            // when
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                process={false} // Treat as a main process, no requestingContainerUuid
                                params={[]}
                                raw={{}}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=conditional-tab]').should('not.exist');
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=default-view]').should('exist').within(() => {
                    cy.contains('No parameters found');
                });
            });
        });

        it('shows main process with raw', () => {
            // when
            const raw = {some: 'data'};
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                process={false} // Treat as a main process, no requestingContainerUuid
                                params={[]}
                                raw={raw}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=conditional-tab]').should('exist');
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=virtual-code-snippet]').should('exist').within(() => {
                    cy.contains(JSON.stringify(raw, null, 2).replace(/\n/g, ''));
                });
            });
        });

        it('shows main process with params', () => {
            // when
            const parameters = [{id: 'someId', label: 'someLabel', value: {display: 'someValue'}}];
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                process={false} // Treat as a main process, no requestingContainerUuid
                                params={parameters}
                                raw={{}}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=conditional-tab]').should('have.length', 2); // Empty raw is shown if parameters are present
                cy.get('tbody').should('exist').within(() => {
                    cy.contains('someId');
                    cy.contains('someLabel');
                    cy.contains('someValue');
                });
            });
        });

        it('shows main process with output collection', () => {
            // when
            const outputCollection = '987654321';
            const parameters = [{id: 'someId', label: 'someLabel', value: {display: 'someValue'}}];

            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.OUTPUT}
                                process={false} // Treat as a main process, no requestingContainerUuid
                                outputUuid={outputCollection}
                                params={parameters}
                                raw={{}}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=conditional-tab]').should('have.length', 3); // Empty raw is shown if parameters are present
                cy.get('tbody').should('exist').within(() => {
                    cy.contains('someId');
                    cy.contains('someLabel');
                    cy.contains('someValue');
                });
            });

            // Visit output tab
            cy.get('[data-cy=conditional-tab]').contains('Collection').should('exist').click();
            cy.get('[data-cy=collection-files-panel]').should('exist');
            cy.get('[data-cy=output-uuid-display]').should('contain', outputCollection);
        });

        // Subprocess

        it('shows subprocess loading', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.INPUT}
                                process={subprocess} // Treat as a subprocess without outputUuid
                                params={null}
                                raw={null}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=conditional-tab]').should('not.exist');
                cy.get('[data-cy=subprocess-circular-progress]').should('exist');
            });
        });

        it('shows subprocess mounts', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const sampleMount = {path: '/', pdh: 'abcdef12abcdef12abcdef12abcdef12+0'};
            cy.mount(
                <Provider store={store}>
                    <MemoryRouter>
                        <StyledEngineProvider injectFirst>
                            <ThemeProvider theme={CustomTheme}>
                                <ProcessIOCard
                                    label={ProcessIOCardType.INPUT}
                                    process={subprocess} // Treat as a subprocess without outputUuid
                                    params={null}
                                    raw={null}
                                    mounts={[sampleMount]}
                                />
                            </ThemeProvider>
                        </StyledEngineProvider>
                    </MemoryRouter>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=subprocess-circular-progress]').should('not.exist');
                cy.getAll('[data-cy=conditional-tab]').should('have.length', 1); // Empty raw is shown if parameters are present
                cy.get('tbody').should('exist').within(() => {
                    cy.contains(sampleMount.pdh);
                });
            });
        });

        it('shows subprocess output collection', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const outputCollection = '123456789';
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.OUTPUT}
                                process={subprocess} // Treat as a subprocess with outputUuid
                                outputUuid={outputCollection}
                                params={null}
                                raw={null}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=conditional-tab]').should('have.length', 1); // Empty raw is shown if parameters are present
                cy.get('[data-cy=output-uuid-display]').should('contain', outputCollection);
            });
        });

        it('shows empty subprocess raw', () => {
            // when
            const subprocess = {containerRequest: {requestingContainerUuid: 'xyz'}};
            const outputCollection = '123456789';
            cy.mount(
                <Provider store={store}>
                    <StyledEngineProvider injectFirst>
                        <ThemeProvider theme={CustomTheme}>
                            <ProcessIOCard
                                label={ProcessIOCardType.OUTPUT}
                                process={subprocess} // Treat as a subprocess with outputUuid
                                outputUuid={outputCollection}
                                params={null}
                                raw={{}}
                            />
                        </ThemeProvider>
                    </StyledEngineProvider>
                </Provider>
                );

            // then
            cy.get('[data-cy=process-io-card]').within(() => {
                cy.get('[data-cy=process-io-circular-progress]').should('not.exist');
                cy.get('[data-cy=conditional-tab]').should('have.length', 2); // Empty raw is shown if parameters are present
                cy.get('[data-cy=conditional-tab]').eq(1).should('exist')
                cy.get('[data-cy=output-uuid-display]').should('contain', outputCollection);
            });
        });

    });
});
