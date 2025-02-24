// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InstanceTypesPanel, calculateKeepBufferOverhead, discountRamByPercent } from './instance-types-panel';
import {
    ThemeProvider,
    StyledEngineProvider,
} from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { combineReducers, createStore } from "redux";
import { Provider } from "react-redux";
import { formatFileSize, formatCWLResourceSize } from 'common/formatters';

describe('<InstanceTypesPanel />', () => {
    let store;

    const initialAuthState = {
        config: {
            clusterConfig: {
                InstanceTypes: {
                    "normalType" : {
                        ProviderType: "provider",
                        Price: 0.123,
                        VCPUs: 6,
                        Preemptible: false,
                        IncludedScratch: 1000,
                        RAM: 5000,
                    },
                    "gpuType" : {
                        ProviderType: "gpuProvider",
                        Price: 0.456,
                        VCPUs: 8,
                        Preemptible: true,
                        IncludedScratch: 500,
                        RAM: 6000,
                        GPU: {
                            DeviceCount: 1,
                            HardwareTarget: '8.6',
                            DriverVersion: '11.4',
                        },
                    },
                },
                Containers: {
                    ReserveExtraRAM: 1000,
                }
            }
        }
    }

    beforeEach(() => {
        store = createStore(combineReducers({
            auth: (state = initialAuthState, action) => state,
        }));
    });

    it('renders instance types', () => {
        // when
        cy.mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <InstanceTypesPanel />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);

        // then
        Object.keys(initialAuthState.config.clusterConfig.InstanceTypes).forEach((instanceKey) => {
            const instanceType = initialAuthState.config.clusterConfig.InstanceTypes[instanceKey];
            cy.get(`[data-cy="${instanceKey}"]`).as('item');

            cy.get('@item').find('h6').contains(instanceKey);
            cy.get('@item').contains(`Provider type${instanceType.ProviderType}`);
            cy.get('@item').contains(`Price$${instanceType.Price}`);
            cy.get('@item').contains(`Cores${instanceType.VCPUs}`);
            cy.get('@item').contains(`Preemptible${instanceType.Preemptible.toString()}`);
            cy.get('@item').contains(`Max disk request${formatCWLResourceSize(instanceType.IncludedScratch)} (${formatFileSize(instanceType.IncludedScratch)})`);
            if (instanceType.GPU && instanceType.GPU.DeviceCount > 0) {
                cy.get('@item').contains(`GPUs${instanceType.GPU.DeviceCount}`);
                cy.get('@item').contains(`Hardware target${instanceType.GPU.HardwareTarget}`);
                cy.get('@item').contains(`Driver version${instanceType.GPU.DriverVersion}`);
            }
        });
    });
});

describe('calculateKeepBufferOverhead', () => {
    it('should calculate correct buffer size', () => {
        const testCases = [
            {input: 0, output: (220<<20)},
            {input: 1, output: (220<<20) + ((1<<26) * (11/10))},
            {input: 2, output: (220<<20) + 2*((1<<26) * (11/10))},
        ];

        for (const {input, output} of testCases) {
            expect(calculateKeepBufferOverhead(input)).to.equal(output);
        }
    });
});

describe('discountRamByPercent', () => {
    it('should inflate ram requirement by 5% of final amount', () => {
        const testCases = [
            {input: 0, output: 0},
            {input: 114, output: 120},
        ];

        for (const {input, output} of testCases) {
            expect(discountRamByPercent(input)).to.equal(output);
        }
    });
});
