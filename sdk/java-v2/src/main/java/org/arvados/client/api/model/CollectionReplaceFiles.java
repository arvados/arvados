/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.HashMap;
import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
public class CollectionReplaceFiles {

    @JsonProperty("collection")
    private CollectionOptions collectionOptions;

    @JsonProperty("replace_files")
    private Map<String, String> replaceFiles;

    public CollectionReplaceFiles() {
        this.collectionOptions = new CollectionOptions();
        this.replaceFiles = new HashMap<>();
    }

    public void addFileReplacement(String targetPath, String sourcePath) {
        this.replaceFiles.put(targetPath, sourcePath);
    }

    @JsonInclude(JsonInclude.Include.NON_NULL)
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class CollectionOptions {
        @JsonProperty("preserve_version")
        private boolean preserveVersion;

        public CollectionOptions() {
            this.preserveVersion = true;
        }

        public boolean isPreserveVersion() {
            return preserveVersion;
        }

        public void setPreserveVersion(boolean preserveVersion) {
            this.preserveVersion = preserveVersion;
        }
    }

    public CollectionOptions getCollectionOptions() {
        return collectionOptions;
    }

    public void setCollectionOptions(CollectionOptions collectionOptions) {
        this.collectionOptions = collectionOptions;
    }

    public Map<String, String> getReplaceFiles() {
        return replaceFiles;
    }

    public void setReplaceFiles(Map<String, String> replaceFiles) {
        this.replaceFiles = replaceFiles;
    }
}