/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.collection;

import org.arvados.client.api.client.GroupsApiClient;
import org.arvados.client.api.client.UsersApiClient;
import org.arvados.client.exception.ArvadosApiException;
import org.arvados.client.api.model.Collection;
import org.arvados.client.common.Patterns;
import org.arvados.client.config.FileConfigProvider;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.exception.ArvadosClientException;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.Optional;

public class CollectionFactory {

    private ConfigProvider config;
    private UsersApiClient usersApiClient;
    private GroupsApiClient groupsApiClient;

    private final String name;
    private final String projectUuid;

    private CollectionFactory(ConfigProvider config, String name, String projectUuid) {
        this.name = name;
        this.projectUuid = projectUuid;
        this.config = config;
        setApiClients();
    }

    public static CollectionFactoryBuilder builder() {
        return new CollectionFactoryBuilder();
    }

    private void setApiClients() {
        if(this.config == null) this.config = new FileConfigProvider();

        this.usersApiClient = new UsersApiClient(config);
        this.groupsApiClient = new GroupsApiClient(config);
    }

    public Collection create() {
        Collection newCollection = new Collection();
        newCollection.setName(getNameOrDefault(name));
        newCollection.setOwnerUuid(getDesiredProjectUuid(projectUuid));

        return newCollection;
    }

    private String getNameOrDefault(String name) {
        return Optional.ofNullable(name).orElseGet(() -> {
            LocalDateTime dateTime = LocalDateTime.now();
            DateTimeFormatter formatter = DateTimeFormatter.ofPattern("Y-MM-dd HH:mm:ss.SSS");
            return String.format("New Collection (%s)", dateTime.format(formatter));
        });
    }

    public String getDesiredProjectUuid(String projectUuid) {
        try {
            if (projectUuid == null || projectUuid.length() == 0){
                return usersApiClient.current().getUuid();
            } else if (projectUuid.matches(Patterns.USER_UUID_PATTERN)) {
                return usersApiClient.get(projectUuid).getUuid();
            } else if (projectUuid.matches(Patterns.GROUP_UUID_PATTERN)) {
                return groupsApiClient.get(projectUuid).getUuid();
            }
        } catch (ArvadosApiException e) {
            throw new ArvadosClientException(String.format("An error occurred while getting project by UUID %s", projectUuid));
        }
        throw new ArvadosClientException(String.format("No project with %s UUID found", projectUuid));
    }

    public static class CollectionFactoryBuilder {
        private ConfigProvider config;
        private String name;
        private String projectUuid;

        CollectionFactoryBuilder() {
        }

        public CollectionFactoryBuilder config(ConfigProvider config) {
            this.config = config;
            return this;
        }

        public CollectionFactoryBuilder name(String name) {
            this.name = name;
            return this;
        }

        public CollectionFactoryBuilder projectUuid(String projectUuid) {
            this.projectUuid = projectUuid;
            return this;
        }

        public CollectionFactory build() {
            return new CollectionFactory(config, name, projectUuid);
        }

    }
}
