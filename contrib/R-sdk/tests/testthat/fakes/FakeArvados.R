# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

FakeArvados <- R6::R6Class(

    "FakeArvados",

    public = list(

        token      = NULL,
        host       = NULL,
        webdavHost = NULL,
        http       = NULL,
        httpParser = NULL,
        REST       = NULL,

        initialize = function(token      = NULL,
                              host       = NULL,
                              webdavHost = NULL,
                              http       = NULL,
                              httpParser = NULL)
        {
            self$token      <- token
            self$host       <- host
            self$webdavHost <- webdavHost
            self$http       <- http
            self$httpParser <- httpParser
        },

        getToken    = function() self$token,
        getHostName = function() self$host,
        getHttpClient = function() self$http,
        getHttpParser = function() self$httpParser,
        getWebDavHostName = function() self$webdavHost
    ),

    cloneable = FALSE
)
