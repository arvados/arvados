/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonPropertyOrder;

import java.time.LocalDateTime;
import java.util.List;

@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
@JsonPropertyOrder({ "command", "container_count", "container_count_max", "container_image", "container_uuid", "cwd", "environment", "expires_at", 
    "filters", "log_uuid", "mounts", "output_name", "output_path", "output_uuid", "output_ttl", "priority", "properties", "requesting_container_uuid", 
    "runtime_constraints", "scheduling_parameters", "state", "use_existing" })
public class Group extends Item {

    @JsonProperty("name")
    private String name;
    @JsonProperty("group_class")
    private String groupClass;
    @JsonProperty("description")
    private String description;
    @JsonProperty(value = "writable_by", access = JsonProperty.Access.WRITE_ONLY)
    private List<String> writableBy;
    @JsonProperty("delete_at")
    private LocalDateTime deleteAt;
    @JsonProperty("trash_at")
    private LocalDateTime trashAt;
    @JsonProperty("is_trashed")
    private Boolean isTrashed;
    @JsonProperty("command")
    private List<String> command;
    @JsonProperty("container_count")
    private Integer containerCount;
    @JsonProperty("container_count_max")
    private Integer containerCountMax;
    @JsonProperty("container_image")
    private String containerImage;
    @JsonProperty("container_uuid")
    private String containerUuid;
    @JsonProperty("cwd")
    private String cwd;
    @JsonProperty("environment")
    private Object environment;
    @JsonProperty("expires_at")
    private LocalDateTime expiresAt;
    @JsonProperty("filters")
    private List<String> filters;
    @JsonProperty("log_uuid")
    private String logUuid;
    @JsonProperty("mounts")
    private Object mounts;
    @JsonProperty("output_name")
    private String outputName;
    @JsonProperty("output_path")
    private String outputPath;
    @JsonProperty("output_uuid")
    private String outputUuid;
    @JsonProperty("output_ttl")
    private Integer outputTtl;
    @JsonProperty("priority")
    private Integer priority;
    @JsonProperty("properties")
    private Object properties;
    @JsonProperty("requesting_container_uuid")
    private String requestingContainerUuid;
    @JsonProperty("runtime_constraints")
    private RuntimeConstraints runtimeConstraints;
    @JsonProperty("scheduling_parameters")
    private Object schedulingParameters;
    @JsonProperty("state")
    private String state;
    @JsonProperty("use_existing")
    private Boolean useExisting;

    public String getName() {
        return this.name;
    }

    public String getGroupClass() {
        return this.groupClass;
    }

    public String getDescription() {
        return this.description;
    }

    public List<String> getWritableBy() {
        return this.writableBy;
    }

    public LocalDateTime getDeleteAt() {
        return this.deleteAt;
    }

    public LocalDateTime getTrashAt() {
        return this.trashAt;
    }

    public Boolean getIsTrashed() {
        return this.isTrashed;
    }

    public List<String> getCommand() {
        return this.command;
    }

    public Integer getContainerCount() {
        return this.containerCount;
    }

    public Integer getContainerCountMax() {
        return this.containerCountMax;
    }

    public String getContainerImage() {
        return this.containerImage;
    }

    public String getContainerUuid() {
        return this.containerUuid;
    }

    public String getCwd() {
        return this.cwd;
    }

    public Object getEnvironment() {
        return this.environment;
    }

    public LocalDateTime getExpiresAt() {
        return this.expiresAt;
    }

    public List<String> getFilters() {
        return this.filters;
    }

    public String getLogUuid() {
        return this.logUuid;
    }

    public Object getMounts() {
        return this.mounts;
    }

    public String getOutputName() {
        return this.outputName;
    }

    public String getOutputPath() {
        return this.outputPath;
    }

    public String getOutputUuid() {
        return this.outputUuid;
    }

    public Integer getOutputTtl() {
        return this.outputTtl;
    }

    public Integer getPriority() {
        return this.priority;
    }

    public Object getProperties() {
        return this.properties;
    }

    public String getRequestingContainerUuid() {
        return this.requestingContainerUuid;
    }

    public RuntimeConstraints getRuntimeConstraints() {
        return this.runtimeConstraints;
    }

    public Object getSchedulingParameters() {
        return this.schedulingParameters;
    }

    public String getState() {
        return this.state;
    }

    public Boolean getUseExisting() {
        return this.useExisting;
    }

    public void setName(String name) {
        this.name = name;
    }

    public void setGroupClass(String groupClass) {
        this.groupClass = groupClass;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public void setWritableBy(List<String> writableBy) {
        this.writableBy = writableBy;
    }

    public void setDeleteAt(LocalDateTime deleteAt) {
        this.deleteAt = deleteAt;
    }

    public void setTrashAt(LocalDateTime trashAt) {
        this.trashAt = trashAt;
    }

    public void setIsTrashed(Boolean isTrashed) {
        this.isTrashed = isTrashed;
    }

    public void setCommand(List<String> command) {
        this.command = command;
    }

    public void setContainerCount(Integer containerCount) {
        this.containerCount = containerCount;
    }

    public void setContainerCountMax(Integer containerCountMax) {
        this.containerCountMax = containerCountMax;
    }

    public void setContainerImage(String containerImage) {
        this.containerImage = containerImage;
    }

    public void setContainerUuid(String containerUuid) {
        this.containerUuid = containerUuid;
    }

    public void setCwd(String cwd) {
        this.cwd = cwd;
    }

    public void setEnvironment(Object environment) {
        this.environment = environment;
    }

    public void setExpiresAt(LocalDateTime expiresAt) {
        this.expiresAt = expiresAt;
    }

    public void setFilters(List<String> filters) {
        this.filters = filters;
    }

    public void setLogUuid(String logUuid) {
        this.logUuid = logUuid;
    }

    public void setMounts(Object mounts) {
        this.mounts = mounts;
    }

    public void setOutputName(String outputName) {
        this.outputName = outputName;
    }

    public void setOutputPath(String outputPath) {
        this.outputPath = outputPath;
    }

    public void setOutputUuid(String outputUuid) {
        this.outputUuid = outputUuid;
    }

    public void setOutputTtl(Integer outputTtl) {
        this.outputTtl = outputTtl;
    }

    public void setPriority(Integer priority) {
        this.priority = priority;
    }

    public void setProperties(Object properties) {
        this.properties = properties;
    }

    public void setRequestingContainerUuid(String requestingContainerUuid) {
        this.requestingContainerUuid = requestingContainerUuid;
    }

    public void setRuntimeConstraints(RuntimeConstraints runtimeConstraints) {
        this.runtimeConstraints = runtimeConstraints;
    }

    public void setSchedulingParameters(Object schedulingParameters) {
        this.schedulingParameters = schedulingParameters;
    }

    public void setState(String state) {
        this.state = state;
    }

    public void setUseExisting(Boolean useExisting) {
        this.useExisting = useExisting;
    }

    public String toString() {
        return "Group(name=" + this.getName() + ", groupClass=" + this.getGroupClass() + ", description=" + this.getDescription() + ", writableBy=" + this.getWritableBy() + ", deleteAt=" + this.getDeleteAt() + ", trashAt=" + this.getTrashAt() + ", isTrashed=" + this.getIsTrashed() + ", command=" + this.getCommand() + ", containerCount=" + this.getContainerCount() + ", containerCountMax=" + this.getContainerCountMax() + ", containerImage=" + this.getContainerImage() + ", containerUuid=" + this.getContainerUuid() + ", cwd=" + this.getCwd() + ", environment=" + this.getEnvironment() + ", expiresAt=" + this.getExpiresAt() + ", filters=" + this.getFilters() + ", logUuid=" + this.getLogUuid() + ", mounts=" + this.getMounts() + ", outputName=" + this.getOutputName() + ", outputPath=" + this.getOutputPath() + ", outputUuid=" + this.getOutputUuid() + ", outputTtl=" + this.getOutputTtl() + ", priority=" + this.getPriority() + ", properties=" + this.getProperties() + ", requestingContainerUuid=" + this.getRequestingContainerUuid() + ", runtimeConstraints=" + this.getRuntimeConstraints() + ", schedulingParameters=" + this.getSchedulingParameters() + ", state=" + this.getState() + ", useExisting=" + this.getUseExisting() + ")";
    }
}
