// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { requirePasswordLogin } from './login-panel';

describe('<LoginPanel />', () => {
    describe('requirePasswordLogin', () => {
        it('should return false if no config specified', () => {
            // given
            const config = null;

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false if no config.clusterConfig specified', () => {
            // given
            const config = {};

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false if no config.clusterConfig.Login specified', () => {
            // given
            const config = {
                clusterConfig: {},
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false if no config.clusterConfig.Login.LDAP and config.clusterConfig.Login.PAM specified', () => {
            // given
            const config = {
                clusterConfig: {
                    Login: {}
                },
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });

        it('should return false if config.clusterConfig.Login.LDAP.Enable and config.clusterConfig.Login.PAM.Enable not specified', () => {
            // given
            const config = {
                clusterConfig: {
                    Login: {
                        PAM: {},
                        LDAP: {},
                    },
                },
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });

        it('should return value from config.clusterConfig.Login.LDAP.Enable', () => {
            // given
            const config = {
                clusterConfig: {
                    Login: {
                        PAM: {},
                        LDAP: {
                            Enable: true
                        },
                    },
                },
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeTruthy();
        });

        it('should return value from config.clusterConfig.Login.PAM.Enable', () => {
            // given
            const config = {
                clusterConfig: {
                    Login: {
                        LDAP: {},
                        PAM: {
                            Enable: true
                        },
                    },
                },
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeTruthy();
        });

        it('should return false for not specified config option config.clusterConfig.Login.NOT_EXISTING.Enable', () => {
            // given
            const config = {
                clusterConfig: {
                    Login: {
                        NOT_EXISTING: {
                            Enable: true
                        },
                    },
                },
            };

            // when
            const result = requirePasswordLogin(config);

            // then
            expect(result).toBeFalsy();
        });
    });
});