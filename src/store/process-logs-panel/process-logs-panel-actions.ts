// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { ProcessLogs } from './process-logs-panel';
import { LogEventType } from 'models/log';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { Dispatch } from 'redux';
import { LogFragment, LogService, logFileToLogType } from 'services/log-service/log-service';
import { Process, getProcess } from 'store/processes/process';
import { navigateTo } from 'store/navigation/navigation-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { CollectionFile, CollectionFileType } from "models/collection-file";

const SNIPLINE = `================ ✀ ================ ✀ ========= Some log(s) were skipped ========= ✀ ================ ✀ ================`;
const LOG_TIMESTAMP_PATTERN = /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{9}Z/;

export const processLogsPanelActions = unionize({
    RESET_PROCESS_LOGS_PANEL: ofType<{}>(),
    INIT_PROCESS_LOGS_PANEL: ofType<{ filters: string[], logs: ProcessLogs }>(),
    SET_PROCESS_LOGS_PANEL_FILTER: ofType<string>(),
    ADD_PROCESS_LOGS_PANEL_ITEM: ofType<ProcessLogs>(),
});

// Max size of logs to fetch in bytes
const maxLogFetchSize: number = 128 * 1000;

type FileWithProgress = {
    file: CollectionFile;
    lastByte: number;
}

export type ProcessLogsPanelAction = UnionOf<typeof processLogsPanelActions>;

export const setProcessLogsPanelFilter = (filter: string) =>
    processLogsPanelActions.SET_PROCESS_LOGS_PANEL_FILTER(filter);

export const initProcessLogsPanel = (processUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, { logService }: ServiceRepository) => {
        try {
            dispatch(processLogsPanelActions.RESET_PROCESS_LOGS_PANEL());
            const process = getProcess(processUuid)(getState().resources);
            if (process?.containerRequest?.uuid) {
                // Get log file size info
                const logFiles = await loadContainerLogFileList(process.containerRequest.uuid, logService);

                // Populate lastbyte 0 for each file
                const filesWithProgress = logFiles.map((file) => ({file, lastByte: 0}));

                // Fetch array of LogFragments
                const logLines = await loadContainerLogFileContents(filesWithProgress, logService, process);

                // Populate initial state with filters
                const initialState = createInitialLogPanelState(logFiles, logLines);
                dispatch(processLogsPanelActions.INIT_PROCESS_LOGS_PANEL(initialState));
            }
        } catch(e) {
            // On error, populate empty state to allow polling to start
            const initialState = createInitialLogPanelState([], []);
            dispatch(processLogsPanelActions.INIT_PROCESS_LOGS_PANEL(initialState));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not load process logs', hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const pollProcessLogs = (processUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, { logService }: ServiceRepository) => {
        try {
            // Get log panel state and process from store
            const currentState = getState().processLogsPanel;
            const process = getProcess(processUuid)(getState().resources);

            // Check if container request is present and initial logs state loaded
            if (process?.containerRequest?.uuid && Object.keys(currentState.logs).length > 0) {
                const logFiles = await loadContainerLogFileList(process.containerRequest.uuid, logService);

                // Determine byte to fetch from while filtering unchanged files
                const filesToUpdateWithProgress = logFiles.reduce((acc, updatedFile) => {
                    // Fetch last byte or 0 for new log files
                    const currentStateLogLastByte = currentState.logs[logFileToLogType(updatedFile)]?.lastByte || 0;

                    const isNew = !Object.keys(currentState.logs).find((currentStateLogName) => (updatedFile.name.startsWith(currentStateLogName)));
                    const isChanged = !isNew && currentStateLogLastByte < updatedFile.size;

                    if (isNew || isChanged) {
                        return acc.concat({file: updatedFile, lastByte: currentStateLogLastByte});
                    } else {
                        return acc;
                    }
                }, [] as FileWithProgress[]);

                // Perform range request(s) for each file
                const logFragments = await loadContainerLogFileContents(filesToUpdateWithProgress, logService, process);

                if (logFragments.length) {
                    // Convert LogFragments to ProcessLogs with All/Main sorting & line-merging
                    const groupedLogs = groupLogs(logFiles, logFragments);
                    await dispatch(processLogsPanelActions.ADD_PROCESS_LOGS_PANEL_ITEM(groupedLogs));
                }
            }
            return Promise.resolve();
        } catch (e) {
            // Remove log when polling error is handled in some way instead of being ignored
            console.log("Polling process logs failed");
            return Promise.reject();
        }
    };

const loadContainerLogFileList = async (containerUuid: string, logService: LogService) => {
    const logCollectionContents = await logService.listLogFiles(containerUuid);

    // Filter only root directory files matching log event types which have bytes
    return logCollectionContents.filter((file): file is CollectionFile => (
        file.type === CollectionFileType.FILE &&
        PROCESS_PANEL_LOG_EVENT_TYPES.indexOf(logFileToLogType(file)) > -1 &&
        file.size > 0
    ));
};

/**
 * Loads the contents of each file from each file's lastByte simultaneously
 *   while respecting the maxLogFetchSize by requesting the start and end
 *   of the desired block and inserting a snipline.
 * @param logFilesWithProgress CollectionFiles with the last byte previously loaded
 * @param logService
 * @param process
 * @returns LogFragment[] containing a single LogFragment corresponding to each input file
 */
const loadContainerLogFileContents = async (logFilesWithProgress: FileWithProgress[], logService: LogService, process: Process) => (
    (await Promise.allSettled(logFilesWithProgress.filter(({file}) => file.size > 0).map(({file, lastByte}) => {
        const requestSize = file.size - lastByte;
        if (requestSize > maxLogFetchSize) {
            const chunkSize = Math.floor(maxLogFetchSize / 2);
            const firstChunkEnd = lastByte+chunkSize-1;
            return Promise.all([
                logService.getLogFileContents(process.containerRequest.uuid, file, lastByte, firstChunkEnd),
                logService.getLogFileContents(process.containerRequest.uuid, file, file.size-chunkSize, file.size-1)
            ] as Promise<(LogFragment)>[]);
        } else {
            return Promise.all([logService.getLogFileContents(process.containerRequest.uuid, file, lastByte, file.size-1)]);
        }
    })).then((res) => {
        if (res.length && res.every(promiseResult => (promiseResult.status === 'rejected'))) {
            // Since allSettled does not pass promise rejection we throw an
            //   error if every request failed
            return Promise.reject("Failed to load logs");
        }
        return res.filter((promiseResult): promiseResult is PromiseFulfilledResult<LogFragment[]> => (
            // Filter out log files with rejected promises
            //   (Promise.all rejects on any failure)
            promiseResult.status === 'fulfilled' &&
            // Filter out files where any fragment is empty
            //   (prevent incorrect snipline generation or an un-resumable situation)
            !!promiseResult.value.every(logFragment => logFragment.contents.length)
        )).map(one => one.value)
    })).map((logResponseSet)=> {
        // For any multi fragment response set, modify the last line of non-final chunks to include a line break and snip line
        //   Don't add snip line as a separate line so that sorting won't reorder it
        for (let i = 1; i < logResponseSet.length; i++) {
            const fragment = logResponseSet[i-1];
            const lastLineIndex = fragment.contents.length-1;
            const lastLineContents = fragment.contents[lastLineIndex];
            const newLastLine = `${lastLineContents}\n${SNIPLINE}`;

            logResponseSet[i-1].contents[lastLineIndex] = newLastLine;
        }

        // Merge LogFragment Array (representing multiple log line arrays) into single LogLine[] / LogFragment
        return logResponseSet.reduce((acc, curr: LogFragment) => ({
            logType: curr.logType,
            contents: [...(acc.contents || []), ...curr.contents]
        }), {} as LogFragment);
    })
);

const createInitialLogPanelState = (logFiles: CollectionFile[], logFragments: LogFragment[]): {filters: string[], logs: ProcessLogs} => {
    const logs = groupLogs(logFiles, logFragments);
    const filters = Object.keys(logs);
    return { filters, logs };
}

/**
 * Converts LogFragments into ProcessLogs, grouping and sorting All/Main logs
 * @param logFiles
 * @param logFragments
 * @returns ProcessLogs for the store
 */
const groupLogs = (logFiles: CollectionFile[], logFragments: LogFragment[]): ProcessLogs => {
    const sortableLogFragments = mergeMultilineLoglines(logFragments);

    const allLogs = mergeSortLogFragments(sortableLogFragments);
    const mainLogs = mergeSortLogFragments(sortableLogFragments.filter((fragment) => (MAIN_EVENT_TYPES.includes(fragment.logType))));

    const groupedLogs = logFragments.reduce((grouped, fragment) => ({
        ...grouped,
        [fragment.logType as string]: {lastByte: fetchLastByteNumber(logFiles, fragment.logType), contents: fragment.contents}
    }), {});

    return {
        [MAIN_FILTER_TYPE]: {lastByte: undefined, contents: mainLogs},
        [ALL_FILTER_TYPE]: {lastByte: undefined, contents: allLogs},
        ...groupedLogs,
    }
};

/**
 * Checks for non-timestamped log lines and merges them with the previous line, assumes they are multi-line logs
 *   If there is no previous line (first line has no timestamp), the line is deleted.
 *   Only used for combined logs that need sorting by timestamp after merging
 * @param logFragments
 * @returns Modified LogFragment[]
 */
const mergeMultilineLoglines = (logFragments: LogFragment[]) => (
    logFragments.map((fragment) => {
        // Avoid altering the original fragment copy
        let fragmentCopy: LogFragment = {
            logType: fragment.logType,
            contents: [...fragment.contents],
        }
        // Merge any non-timestamped lines in sortable log types with previous line
        if (fragmentCopy.contents.length && !NON_SORTED_LOG_TYPES.includes(fragmentCopy.logType)) {
            for (let i = 0; i < fragmentCopy.contents.length; i++) {
                const lineContents = fragmentCopy.contents[i];
                if (!lineContents.match(LOG_TIMESTAMP_PATTERN)) {
                    // Partial line without timestamp detected
                    if (i > 0) {
                        // If not first line, copy line to previous line
                        const previousLineContents = fragmentCopy.contents[i-1];
                        const newPreviousLineContents = `${previousLineContents}\n${lineContents}`;
                        fragmentCopy.contents[i-1] = newPreviousLineContents;
                    }
                    // Delete the current line and prevent iterating
                    fragmentCopy.contents.splice(i, 1);
                    i--;
                }
            }
        }
        return fragmentCopy;
    })
);

/**
 * Merges log lines of different types and sorts types that contain timestamps (are sortable)
 * @param logFragments
 * @returns string[] of merged and sorted log lines
 */
const mergeSortLogFragments = (logFragments: LogFragment[]): string[] => {
    const sortableLines = fragmentsToLines(logFragments
        .filter((fragment) => (!NON_SORTED_LOG_TYPES.includes(fragment.logType))));

    const nonSortableLines = fragmentsToLines(logFragments
        .filter((fragment) => (NON_SORTED_LOG_TYPES.includes(fragment.logType)))
        .sort((a, b) => (a.logType.localeCompare(b.logType))));

    return [...nonSortableLines, ...sortableLines.sort(sortLogLines)]
};

const sortLogLines = (a: string, b: string) => {
    return a.localeCompare(b);
};

const fragmentsToLines = (fragments: LogFragment[]): string[] => (
    fragments.reduce((acc, fragment: LogFragment) => (
        acc.concat(...fragment.contents)
    ), [] as string[])
);

const fetchLastByteNumber = (logFiles: CollectionFile[], key: string) => {
    return logFiles.find((file) => (file.name.startsWith(key)))?.size
};

export const navigateToLogCollection = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.collectionService.get(uuid);
            dispatch<any>(navigateTo(uuid));
        } catch {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Could not request collection', hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

const ALL_FILTER_TYPE = 'All logs';

const MAIN_FILTER_TYPE = 'Main logs';
const MAIN_EVENT_TYPES = [
    LogEventType.CRUNCH_RUN,
    LogEventType.STDERR,
    LogEventType.STDOUT,
];

const PROCESS_PANEL_LOG_EVENT_TYPES = [
    LogEventType.ARV_MOUNT,
    LogEventType.CRUNCH_RUN,
    LogEventType.CRUNCHSTAT,
    LogEventType.DISPATCH,
    LogEventType.HOSTSTAT,
    LogEventType.NODE_INFO,
    LogEventType.STDERR,
    LogEventType.STDOUT,
    LogEventType.CONTAINER,
    LogEventType.KEEPSTORE,
];

const NON_SORTED_LOG_TYPES = [
    LogEventType.NODE_INFO,
    LogEventType.CONTAINER,
];
