/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.logic.keep;

import com.google.common.collect.Lists;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.io.FileUtils;
import org.arvados.client.api.client.KeepServicesApiClient;
import org.arvados.client.api.model.KeepService;
import org.arvados.client.api.model.KeepServiceList;
import org.arvados.client.common.Characters;
import org.arvados.client.common.Headers;
import org.arvados.client.config.ConfigProvider;
import org.arvados.client.exception.ArvadosApiException;
import org.arvados.client.exception.ArvadosClientException;
import org.slf4j.Logger;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.function.Function;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class KeepClient {

    private final KeepServicesApiClient keepServicesApiClient;
    private final Logger log = org.slf4j.LoggerFactory.getLogger(KeepClient.class);
    private List<KeepService> keepServices;
    private List<KeepService> writableServices;
    private Map<String, KeepService> gatewayServices;
    private Integer maxReplicasPerService;
    private final ConfigProvider config;

    public KeepClient(ConfigProvider config) {
        this.config = config;
        keepServicesApiClient = new KeepServicesApiClient(config);
    }

    public byte[] getDataChunk(KeepLocator keepLocator) {

        Map<String, String> headers = new HashMap<>();
        Map<String, FileTransferHandler> rootsMap = new HashMap<>();

        List<String> sortedRoots = mapNewServices(rootsMap, keepLocator, false, false, headers);

        byte[] dataChunk = sortedRoots
                .stream()
                .map(rootsMap::get)
                .map(r -> r.get(keepLocator))
                .filter(Objects::nonNull)
                .findFirst()
                .orElse(null);

        if (dataChunk == null) {
            throw new ArvadosClientException("No server responding. Unable to download data chunk.");
        }

        return dataChunk;
    }

    public String put(File data, int copies, int numRetries) {

        byte[] fileBytes;
        try {
            fileBytes = FileUtils.readFileToByteArray(data);
        } catch (IOException e) {
            throw new ArvadosClientException("An error occurred while reading data chunk", e);
        }

        String dataHash = DigestUtils.md5Hex(fileBytes);
        String locatorString = String.format("%s+%d", dataHash, data.length());

        if (copies < 1) {
            return locatorString;
        }
        KeepLocator locator = new KeepLocator(locatorString);

        // Tell the proxy how many copies we want it to store
        Map<String, String> headers = new HashMap<>();
        headers.put(Headers.X_KEEP_DESIRED_REPLICAS, String.valueOf(copies));

        Map<String, FileTransferHandler> rootsMap = new HashMap<>();
        List<String> sortedRoots = mapNewServices(rootsMap, locator, false, true, headers);

        int numThreads = 0;
        if (maxReplicasPerService == null || maxReplicasPerService >= copies) {
            numThreads = 1;
        } else {
            numThreads = ((Double) Math.ceil(1.0 * copies / maxReplicasPerService)).intValue();
        }
        log.debug("Pool max threads is {}", numThreads);

        List<CompletableFuture<String>> futures = Lists.newArrayList();
        for (int i = 0; i < numThreads; i++) {
            String root = sortedRoots.get(i);
            FileTransferHandler keepServiceLocal = rootsMap.get(root);
            futures.add(CompletableFuture.supplyAsync(() -> keepServiceLocal.put(dataHash, data)));
        }

        @SuppressWarnings("unchecked")
        CompletableFuture<String>[] array = futures.toArray(new CompletableFuture[0]);

        return Stream.of(array)
                .map(CompletableFuture::join)
                .reduce((a, b) -> b)
                .orElse(null);
    }

    private List<String> mapNewServices(Map<String, FileTransferHandler> rootsMap, KeepLocator locator,
                                        boolean forceRebuild, boolean needWritable, Map<String, String> headers) {

        headers.putIfAbsent("Authorization", String.format("OAuth2 %s", config.getApiToken()));
        List<String> localRoots = weightedServiceRoots(locator, forceRebuild, needWritable);
        for (String root : localRoots) {
            FileTransferHandler keepServiceLocal = new FileTransferHandler(root, headers, config);
            rootsMap.putIfAbsent(root, keepServiceLocal);
        }
        return localRoots;
    }

    /**
     * Return an array of Keep service endpoints, in the order in which they should be probed when reading or writing
     * data with the given hash+hints.
     */
    private List<String> weightedServiceRoots(KeepLocator locator, boolean forceRebuild, boolean needWritable) {

        buildServicesList(forceRebuild);

        List<String> sortedRoots = new ArrayList<>();

        // Use the services indicated by the given +K@... remote
        // service hints, if any are present and can be resolved to a
        // URI.
        //
        for (String hint : locator.getHints()) {
            if (hint.startsWith("K@")) {
                if (hint.length() == 7) {
                    sortedRoots.add(String.format("https://keep.%s.arvadosapi.com/", hint.substring(2)));
                } else if (hint.length() == 29) {
                    KeepService svc = gatewayServices.get(hint.substring(2));
                    if (svc != null) {
                        sortedRoots.add(svc.getServiceRoot());
                    }
                }
            }
        }

        // Sort the available local services by weight (heaviest first)
        // for this locator, and return their service_roots (base URIs)
        // in that order.
        List<KeepService> useServices = keepServices;
        if (needWritable) {
            useServices = writableServices;
        }
        anyNonDiskServices(useServices);

        sortedRoots.addAll(useServices
                .stream()
                .sorted((ks1, ks2) -> serviceWeight(locator.getMd5sum(), ks2.getUuid())
                        .compareTo(serviceWeight(locator.getMd5sum(), ks1.getUuid())))
                .map(KeepService::getServiceRoot)
                .collect(Collectors.toList()));

        return sortedRoots;
    }

    private void buildServicesList(boolean forceRebuild) {
        if (keepServices != null && !forceRebuild) {
            return;
        }
        KeepServiceList keepServiceList;
        try {
            keepServiceList = keepServicesApiClient.accessible();
        } catch (ArvadosApiException e) {
            throw new ArvadosClientException("Cannot obtain list of accessible keep services");
        }
        // Gateway services are only used when specified by UUID,
        // so there's nothing to gain by filtering them by
        // service_type.
        gatewayServices = keepServiceList.getItems().stream().collect(Collectors.toMap(KeepService::getUuid, Function.identity()));

        if (gatewayServices.isEmpty()) {
            throw new ArvadosClientException("No gateway services available!");
        }

        // Precompute the base URI for each service.
        for (KeepService keepService : gatewayServices.values()) {
            String serviceHost = keepService.getServiceHost();
            if (!serviceHost.startsWith("[") && serviceHost.contains(Characters.COLON)) {
                // IPv6 URIs must be formatted like http://[::1]:80/...
                serviceHost = String.format("[%s]", serviceHost);
            }

            String protocol = keepService.getServiceSslFlag() ? "https" : "http";
            String serviceRoot = String.format("%s://%s:%d/", protocol, serviceHost, keepService.getServicePort());
            keepService.setServiceRoot(serviceRoot);
        }

        keepServices = gatewayServices.values().stream().filter(ks -> !ks.getServiceType().startsWith("gateway:")).collect(Collectors.toList());
        writableServices = keepServices.stream().filter(ks -> !ks.getReadOnly()).collect(Collectors.toList());

        // For disk type services, max_replicas_per_service is 1
        // It is unknown (unlimited) for other service types.
        if (anyNonDiskServices(writableServices)) {
            maxReplicasPerService = null;
        } else {
            maxReplicasPerService = 1;
        }
    }

    private Boolean anyNonDiskServices(List<KeepService> useServices) {
        return useServices.stream().anyMatch(ks -> !ks.getServiceType().equals("disk"));
    }

    /**
     * Compute the weight of a Keep service endpoint for a data block with a known hash.
     * <p>
     * The weight is md5(h + u) where u is the last 15 characters of the service endpoint's UUID.
     */
    private static String serviceWeight(String dataHash, String serviceUuid) {
        String shortenedUuid;
        if (serviceUuid != null && serviceUuid.length() >= 15) {
            int substringIndex = serviceUuid.length() - 15;
            shortenedUuid = serviceUuid.substring(substringIndex);
        } else {
            shortenedUuid = (serviceUuid == null) ? "" : serviceUuid;
        }
        return DigestUtils.md5Hex(dataHash + shortenedUuid);
    }

}
