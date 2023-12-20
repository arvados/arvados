// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { configure, mount } from "enzyme";
import { InstanceTypesPanel, calculateKeepBufferOverhead, discountRamByPercent } from './instance-types-panel';
import Adapter from "enzyme-adapter-react-16";
import { combineReducers, createStore } from "redux";
import { Provider } from "react-redux";
import { formatFileSize, formatCWLResourceSize } from 'common/formatters';

configure({ adapter: new Adapter() });

describe('<InstanceTypesPanel />', () => {

    // let props;
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
                        CUDA: {
                            DeviceCount: 1,
                            HardwareCapability: '8.6',
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
            auth: (state: any = initialAuthState, action: any) => state,
        }));
    });

    it('renders instance types', () => {
        // when
        const panel = mount(
            <Provider store={store}>
                <InstanceTypesPanel />
            </Provider>);

        // then
        Object.keys(initialAuthState.config.clusterConfig.InstanceTypes).forEach((instanceKey) => {
            const instanceType = initialAuthState.config.clusterConfig.InstanceTypes[instanceKey];
            const item = panel.find(`Grid[data-cy="${instanceKey}"]`)

            expect(item.find('h6').text()).toContain(instanceKey);
            expect(item.text()).toContain(`Provider type${instanceType.ProviderType}`);
            expect(item.text()).toContain(`Price$${instanceType.Price}`);
            expect(item.text()).toContain(`Cores${instanceType.VCPUs}`);
            expect(item.text()).toContain(`Preemptible${instanceType.Preemptible.toString()}`);
            expect(item.text()).toContain(`Max disk request${formatCWLResourceSize(instanceType.IncludedScratch)} (${formatFileSize(instanceType.IncludedScratch)})`);
            if (instanceType.CUDA && instanceType.CUDA.DeviceCount > 0) {
                expect(item.text()).toContain(`CUDA GPUs${instanceType.CUDA.DeviceCount}`);
                expect(item.text()).toContain(`Hardware capability${instanceType.CUDA.HardwareCapability}`);
                expect(item.text()).toContain(`Driver version${instanceType.CUDA.DriverVersion}`);
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
            expect(calculateKeepBufferOverhead(input)).toBe(output);
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
            expect(discountRamByPercent(input)).toBe(output);
        }
    });
});
