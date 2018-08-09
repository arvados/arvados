# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

context("Http Parser")


test_that("parseJSONResponse generates and returns JSON object from server response", {

    JSONContent <- "{\"bar\":{\"foo\":[10]}}"
    serverResponse <- list()
    serverResponse$content <- charToRaw(JSONContent)
    serverResponse$headers[["Content-Type"]] <- "application/json; charset=utf-8"
    class(serverResponse) <- c("response")

    parser <- HttpParser$new()

    result <- parser$parseJSONResponse(serverResponse)
    barExists <- !is.null(result$bar)

    expect_that(barExists, is_true())
    expect_that(unlist(result$bar$foo), equals(10))
})

test_that(paste("parseResponse generates and returns character vector",
                "from server response if outputType is text"), {

    content <- "random text"
    serverResponse <- list()
    serverResponse$content <- charToRaw(content)
    serverResponse$headers[["Content-Type"]] <- "text/plain; charset=utf-8"
    class(serverResponse) <- c("response")

    parser <- HttpParser$new()
    parsedResponse <- parser$parseResponse(serverResponse, "text")

    expect_that(parsedResponse, equals("random text"))
})


webDAVResponseSample =
    paste0("<?xml version=\"1.0\" encoding=\"UTF-8\"?><D:multistatus xmlns:",
           "D=\"DAV:\"><D:response><D:href>/c=aaaaa-bbbbb-ccccccccccccccc</D",
           ":href><D:propstat><D:prop><D:resourcetype><D:collection xmlns:D=",
           "\"DAV:\"/></D:resourcetype><D:getlastmodified>Fri, 11 Jan 2018 1",
           "1:11:11 GMT</D:getlastmodified><D:displayname></D:displayname><D",
           ":supportedlock><D:lockentry xmlns:D=\"DAV:\"><D:lockscope><D:exc",
           "lusive/></D:lockscope><D:locktype><D:write/></D:locktype></D:loc",
           "kentry></D:supportedlock></D:prop><D:status>HTTP/1.1 200 OK</D:s",
           "tatus></D:propstat></D:response><D:response><D:href>/c=aaaaa-bbb",
           "bb-ccccccccccccccc/myFile.exe</D:href><D:propstat><D:prop><D:r",
           "esourcetype></D:resourcetype><D:getlastmodified>Fri, 12 Jan 2018",
           " 22:22:22 GMT</D:getlastmodified><D:getcontenttype>text/x-c++src",
           "; charset=utf-8</D:getcontenttype><D:displayname>myFile.exe</D",
           ":displayname><D:getcontentlength>25</D:getcontentlength><D:getet",
           "ag>\"123b12dd1234567890\"</D:getetag><D:supportedlock><D:lockent",
           "ry xmlns:D=\"DAV:\"><D:lockscope><D:exclusive/></D:lockscope><D:",
           "locktype><D:write/></D:locktype></D:lockentry></D:supportedlock>",
           "</D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:re",
           "sponse></D:multistatus>")



test_that(paste("getFileNamesFromResponse returns file names belonging to specific",
                "collection parsed from webDAV server response"), {

    serverResponse <- list()
    serverResponse$content <- charToRaw(webDAVResponseSample)
    serverResponse$headers[["Content-Type"]] <- "text/xml; charset=utf-8"
    class(serverResponse) <- c("response")
    url <- URLencode("https://webdav/c=aaaaa-bbbbb-ccccccccccccccc")

    parser <- HttpParser$new()
    result <- parser$getFileNamesFromResponse(serverResponse, url)
    expectedResult <- "myFile.exe"
    resultMatchExpected <- all.equal(result, expectedResult)

    expect_that(resultMatchExpected, is_true())
})

test_that(paste("getFileSizesFromResponse returns file sizes",
                "parsed from webDAV server response"), {

    serverResponse <- list()
    serverResponse$content <- charToRaw(webDAVResponseSample)
    serverResponse$headers[["Content-Type"]] <- "text/xml; charset=utf-8"
    class(serverResponse) <- c("response")
    url <- URLencode("https://webdav/c=aaaaa-bbbbb-ccccccccccccccc")

    parser <- HttpParser$new()
    expectedResult <- "25"
    result <- parser$getFileSizesFromResponse(serverResponse, url)
    resultMatchExpected <- result == expectedResult

    expect_that(resultMatchExpected, is_true())
})
