/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import com.google.common.collect.ImmutableList;
import org.arvados.client.common.Characters;

import java.io.File;
import java.util.Collection;
import java.util.List;
import java.util.stream.Collectors;

public class ManifestFactory {

    private Collection<File> files;
    private List<String> locators;

    ManifestFactory(Collection<File> files, List<String> locators) {
        this.files = files;
        this.locators = locators;
    }

    public static ManifestFactoryBuilder builder() {
        return new ManifestFactoryBuilder();
    }

    public String create() {
        ImmutableList.Builder<String> builder = new ImmutableList.Builder<String>()
                .add(Characters.DOT)
                .addAll(locators);
        long filePosition = 0;
        for (File file : files) {
            builder.add(String.format("%d:%d:%s", filePosition, file.length(), file.getName().replace(" ", Characters.SPACE)));
            filePosition += file.length();
        }
        String manifest = builder.build().stream().collect(Collectors.joining(" ")).concat(Characters.NEW_LINE);
        return manifest;
    }

    public static class ManifestFactoryBuilder {
        private Collection<File> files;
        private List<String> locators;

        ManifestFactoryBuilder() {
        }

        public ManifestFactory.ManifestFactoryBuilder files(Collection<File> files) {
            this.files = files;
            return this;
        }

        public ManifestFactory.ManifestFactoryBuilder locators(List<String> locators) {
            this.locators = locators;
            return this;
        }

        public ManifestFactory build() {
            return new ManifestFactory(files, locators);
        }

    }
}
