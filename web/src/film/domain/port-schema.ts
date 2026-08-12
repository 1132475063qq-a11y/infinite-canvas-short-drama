import type { CanvasConnection, CanvasNodeData } from "@/types/canvas";

import type { FilmNodeKind, FilmNodePort, FilmPortSchema, FilmPortType } from "./types";

type PortInput = Omit<FilmPortSchema, "direction" | "accepts"> & { accepts: readonly FilmPortType[] };
type PortOutput = Omit<FilmPortSchema, "direction" | "accepts">;

const input = ({ accepts, ...port }: PortInput): FilmPortSchema => ({ ...port, direction: "input", accepts });
const output = (port: PortOutput): FilmPortSchema => ({ ...port, direction: "output", accepts: [] });

const optionalInput = (id: FilmNodePort, type: FilmPortType, role: string, accepts: readonly FilmPortType[], multiple = false) =>
    input({ id, type, role, accepts, required: false, multiple });
const requiredInput = (id: FilmNodePort, type: FilmPortType, role: string, accepts: readonly FilmPortType[], multiple = false) =>
    input({ id, type, role, accepts, required: true, multiple });
const productionOutput = (id: FilmNodePort, type: FilmPortType, role: string, multiple = true) =>
    output({ id, type, role, required: false, multiple });

/**
 * The schema is exhaustive even before every Film node has a renderer. This keeps
 * the production contract stable while Phase 4 adds node UI incrementally.
 */
export const FILM_NODE_PORT_SCHEMAS = {
    project: [productionOutput("scene_context", "scene", "project_context")],
    story: [optionalInput("scene_context", "story", "project_context", ["scene"]), productionOutput("scene_context", "story", "story_context")],
    script: [optionalInput("scene_context", "story", "story_context", ["story", "scene"]), productionOutput("shot_contract", "script", "script_contract")],
    scene: [optionalInput("scene_context", "scene", "project_context", ["story", "script", "scene"]), productionOutput("scene_context", "scene", "scene_context")],
    character: [productionOutput("asset_reference", "character", "character_reference")],
    character_state: [requiredInput("asset_reference", "character", "character_identity", ["character"]), productionOutput("asset_reference", "character", "character_state_reference")],
    location: [productionOutput("asset_reference", "location", "location_reference")],
    location_view: [requiredInput("asset_reference", "location", "location_identity", ["location"]), productionOutput("asset_reference", "location", "location_view_reference")],
    prop: [productionOutput("asset_reference", "prop", "prop_reference")],
    voice_profile: [productionOutput("asset_reference", "audio", "voice_reference")],
    acting: [optionalInput("shot_contract", "script", "shot_contract", ["script"]), optionalInput("acting_direction", "acting", "acting_context", ["scene", "script", "acting"]), productionOutput("acting_direction", "acting", "acting_direction")],
    acting_profile: [productionOutput("acting_direction", "acting", "acting_profile")],
    scene_acting: [requiredInput("scene_context", "scene", "scene_context", ["scene"]), optionalInput("acting_direction", "acting", "acting_profile", ["acting"], true), productionOutput("acting_direction", "acting", "scene_acting")],
    storyboard: [requiredInput("shot_contract", "script", "script_contract", ["script"]), productionOutput("shot_contract", "script", "storyboard_contract")],
    shot: [
        requiredInput("scene_context", "scene", "scene_context", ["scene"]),
        optionalInput("shot_contract", "script", "storyboard_contract", ["script"]),
        optionalInput("asset_reference", "reference", "production_reference", ["character", "location", "prop", "audio", "reference"], true),
        optionalInput("acting_direction", "acting", "scene_acting", ["acting"]),
        productionOutput("shot_contract", "script", "shot_contract"),
    ],
    shot_reference_pack: [requiredInput("asset_reference", "reference", "shot_reference", ["character", "location", "prop", "audio", "reference"], true), productionOutput("asset_reference", "reference", "reference_pack")],
    prompt_pack: [
        requiredInput("shot_contract", "script", "shot_contract", ["script"]),
        optionalInput("asset_reference", "reference", "prompt_reference", ["character", "location", "prop", "audio", "reference"], true),
        optionalInput("acting_direction", "acting", "acting_direction", ["acting"]),
        productionOutput("prompt", "prompt", "compiled_prompt"),
    ],
    generation: [requiredInput("prompt", "prompt", "compiled_prompt", ["prompt"]), optionalInput("retry_request", "decision", "retry_request", ["decision"]), productionOutput("generation_job", "generation", "generation_job")],
    generation_attempt: [requiredInput("generation_job", "generation", "generation_job", ["generation"]), productionOutput("generation_job", "generation", "generation_attempt")],
    result: [requiredInput("generation_job", "generation", "generation_attempt", ["generation"]), productionOutput("result", "artifact", "generated_result")],
    qc: [requiredInput("result", "artifact", "generated_result", ["artifact", "image", "video", "audio"]), optionalInput("shot_contract", "script", "shot_contract", ["script"]), optionalInput("scene_context", "continuity", "continuity_context", ["continuity"]), productionOutput("qc_decision", "qc", "qc_decision")],
    continuity: [optionalInput("scene_context", "continuity", "continuity_in", ["scene", "continuity"], true), productionOutput("scene_context", "continuity", "continuity_out")],
    retry: [requiredInput("qc_decision", "qc", "qc_decision", ["qc", "decision"]), optionalInput("retry_request", "decision", "human_retry_request", ["decision"]), productionOutput("retry_request", "decision", "retry_request")],
    needs_you: [optionalInput("qc_decision", "decision", "human_decision_input", ["qc", "decision", "artifact"], true), productionOutput("generic", "decision", "human_decision")],
    agent_task: [optionalInput("generic", "artifact", "agent_input", ["artifact", "decision", "script", "scene", "reference"], true), productionOutput("generic", "artifact", "agent_output")],
    delivery: [requiredInput("result", "artifact", "approved_result", ["artifact", "image", "video", "audio"], true), productionOutput("result", "artifact", "delivery_artifact")],
} as const satisfies Record<FilmNodeKind, readonly FilmPortSchema[]>;

export function filmPortSchemas(kind: FilmNodeKind): readonly FilmPortSchema[] {
    return FILM_NODE_PORT_SCHEMAS[kind];
}

export function findFilmPortSchema(kind: FilmNodeKind, portId: FilmNodePort, direction: FilmPortSchema["direction"]): FilmPortSchema | undefined {
    return filmPortSchemas(kind).find((port) => port.id === portId && port.direction === direction);
}

export function areFilmPortSchemasCompatible(outputPort: FilmPortSchema, inputPort: FilmPortSchema): boolean {
    return outputPort.direction === "output" && inputPort.direction === "input" && inputPort.accepts.includes(outputPort.type);
}

export function missingRequiredFilmInputPorts(node: CanvasNodeData, connections: readonly CanvasConnection[]): FilmPortSchema[] {
    if (!node.filmKind) return [];
    const connectedPorts = new Set(connections.filter((connection) => connection.toNodeId === node.id).map((connection) => connection.filmPorts?.to).filter(Boolean));
    return filmPortSchemas(node.filmKind).filter((port) => port.direction === "input" && port.required && !connectedPorts.has(port.id));
}

export function filmInputPortHasCapacity(node: CanvasNodeData, port: FilmPortSchema, connections: readonly CanvasConnection[]): boolean {
    if (port.direction !== "input" || port.multiple) return true;
    return !connections.some((connection) => connection.toNodeId === node.id && connection.filmPorts?.to === port.id);
}
