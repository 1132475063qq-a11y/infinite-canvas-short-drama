import { create } from "zustand";
import { persist, type PersistStorage, type StorageValue } from "zustand/middleware";

import { nanoid } from "nanoid";
import { localForageStorage } from "@/lib/localforage-storage";
import type { CanvasBackgroundMode } from "@/lib/canvas-theme";
import { deriveCanvasDocumentGroups, migrateCanvasProjectDocument } from "@/film/domain/document-migration";
import type { CanvasAssistantSession, CanvasConnection, CanvasNodeData, CanvasProjectDocument, ViewportTransform } from "@/types/canvas";
import type { DirectorScene } from "@/types/director";
import type { TimelineProject } from "@/types/timeline";

export type CanvasProject = CanvasProjectDocument & {
    id: string;
    projectId?: string;
    title: string;
    createdAt: string;
    updatedAt: string;
    connections: CanvasConnection[];
    chatSessions: CanvasAssistantSession[];
    activeChatId: string | null;
    backgroundMode: CanvasBackgroundMode;
    showImageInfo: boolean;
    viewport: ViewportTransform;
    directorScenes: DirectorScene[];
    timeline?: TimelineProject;
};

type CanvasStore = {
    hydrated: boolean;
    projects: CanvasProject[];
    createProject: (title?: string, projectId?: string) => string;
    importProject: (project: Partial<CanvasProject>) => string;
    openProject: (id: string) => CanvasProject | null;
    renameProject: (id: string, title: string) => void;
    deleteProjects: (ids: string[]) => void;
    replaceProjects: (projects: CanvasProject[]) => void;
    updateProject: (id: string, patch: Partial<Pick<CanvasProject, "projectId" | "schemaVersion" | "layout" | "nodes" | "connections" | "groups" | "chatSessions" | "activeChatId" | "backgroundMode" | "showImageInfo" | "viewport" | "directorScenes" | "timeline">>) => void;
};

const initialViewport: ViewportTransform = { x: 0, y: 0, k: 1 };
export const CANVAS_STORE_KEY = "infinite-canvas:canvas_store";
type PersistedCanvasState = Pick<CanvasStore, "projects">;
let saveTimer: ReturnType<typeof setTimeout> | null = null;
let queuedPersistState: PersistedCanvasState | null = null;
let queuedPersistName = CANVAS_STORE_KEY;
let queuedPersistValue: string | null = null;

const canvasStorage: PersistStorage<CanvasStore> = {
    getItem: async (name) => {
        const value = await localForageStorage.getItem(name);
        if (!value) return null;
        const parsed = JSON.parse(value) as StorageValue<CanvasStore>;
        queuedPersistState = parsed.state as PersistedCanvasState;
        return parsed;
    },
    setItem: (name, value) => {
        const nextState = value.state as PersistedCanvasState;
        if (queuedPersistState && queuedPersistState.projects === nextState.projects) return;
        queuedPersistState = nextState;
        queuedPersistName = name;
        queuedPersistValue = JSON.stringify(value);
        if (saveTimer) clearTimeout(saveTimer);
        saveTimer = setTimeout(() => {
            saveTimer = null;
            const payload = queuedPersistValue;
            queuedPersistValue = null;
            if (payload) void localForageStorage.setItem(queuedPersistName, payload);
        }, 400);
    },
    removeItem: (name) => localForageStorage.removeItem(name),
};

export async function flushCanvasStorePersistence() {
    if (saveTimer) {
        clearTimeout(saveTimer);
        saveTimer = null;
    }
    const payload = queuedPersistValue;
    queuedPersistValue = null;
    if (payload) await localForageStorage.setItem(queuedPersistName, payload);
}

export const useCanvasStore = create<CanvasStore>()(
    persist(
        (set, get) => ({
            hydrated: false,
            projects: [],
            createProject: (title = "未命名画布", projectId) => {
                const now = new Date().toISOString();
                const id = nanoid();
                const project: CanvasProject = {
                    id,
                    schemaVersion: 2,
                    layout: { gridSize: 8 },
                    projectId,
                    title,
                    createdAt: now,
                    updatedAt: now,
                    nodes: [],
                    connections: [],
                    groups: [],
                    chatSessions: [],
                    activeChatId: null,
                    backgroundMode: "dots",
                    showImageInfo: false,
                    viewport: initialViewport,
                    directorScenes: [],
                };
                set((state) => ({ projects: [project, ...state.projects] }));
                return id;
            },
            importProject: (source) => {
                const now = new Date().toISOString();
                const project: CanvasProject = {
                    id: nanoid(),
                    schemaVersion: 2,
                    layout: source.layout || { gridSize: 8 },
                    projectId: source.projectId,
                    title: source.title || "导入画布",
                    createdAt: source.createdAt || now,
                    updatedAt: now,
                    nodes: source.nodes || [],
                    connections: source.connections || [],
                    groups: deriveCanvasDocumentGroups(source.nodes || [], source.groups || []),
                    chatSessions: source.chatSessions || [],
                    activeChatId: source.activeChatId || null,
                    backgroundMode: source.backgroundMode || "dots",
                    showImageInfo: source.showImageInfo || false,
                    viewport: source.viewport || initialViewport,
                    directorScenes: source.directorScenes || [],
                };
                set((state) => ({ projects: [project, ...state.projects] }));
                return project.id;
            },
            openProject: (id) => {
                return get().projects.find((item) => item.id === id) || null;
            },
            renameProject: (id, title) =>
                set((state) => ({
                    projects: state.projects.map((project) => (project.id === id ? { ...project, title: title.trim() || project.title, updatedAt: new Date().toISOString() } : project)),
                })),
            deleteProjects: (ids) =>
                set((state) => {
                    const projects = state.projects.filter((project) => !ids.includes(project.id));
                    return { projects };
                }),
            replaceProjects: (projects) => set({ projects: projects.map((project) => migrateCanvasProjectDocument(project)) }),
            updateProject: (id, patch) =>
                set((state) => ({
                    projects: state.projects.map((project) => {
                        if (project.id !== id) return project;
                        const nodes = patch.nodes || project.nodes;
                        return {
                            ...project,
                            ...patch,
                            groups: patch.groups || (patch.nodes ? deriveCanvasDocumentGroups(nodes, project.groups) : project.groups),
                            updatedAt: new Date().toISOString(),
                        };
                    }),
                })),
        }),
        {
            name: CANVAS_STORE_KEY,
            storage: canvasStorage,
            partialize: (state) =>
                ({
                    projects: state.projects,
                }) as StorageValue<CanvasStore>["state"],
            onRehydrateStorage: () => (state) => {
                if (state?.projects) state.replaceProjects(state.projects);
                useCanvasStore.setState({ hydrated: true });
            },
        },
    ),
);
